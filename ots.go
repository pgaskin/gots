// Package gots is a WebAssembly wrapper for the OTS library.
package gots

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type inst struct {
	module      api.Module
	index       uint32
	tableAction TableActionFunc
	message     MessageFunc
	maxSize     int
}

// Option specifies an option for processing fonts.
type Option func(*inst)

// WithIndex specifies the font index to extract from a font collection. By
// default, the whole collection will be returned. Ignored if not a font
// collection.
func WithIndex(idx uint32) Option {
	return func(is *inst) {
		is.index = idx
	}
}

// Tag describes a SFNT table,
type Tag [4]uint8

// TableAction is the action for OTS to take when processing a table.
//
// This matches the ots::TableAction enum.
type TableAction int

const (
	// TableActionDefault uses OTS's default action for that table.
	TableActionDefault TableAction = iota
	// TableActionSanitize sanitizes the table, potentially dropping it.
	TableActionSanitize
	// TableActionPassThru serializes the table unchanged.
	TableActionPassThru
	// TableActionDrop drops the table.
	TableActionDrop
)

// TableActionFunc determines which action to take when processing the specified
// table.
type TableActionFunc func(table Tag) TableAction

// WithTableAction specifies a function which is called to determine what to do
// with each font table. By default, TableActionDefault is returned for all
// tables.
func WithTableAction(fn TableActionFunc) Option {
	return func(is *inst) {
		if fn != nil {
			is.tableAction = fn
		}
	}
}

// MessageLevel is a log message level for OTS.
//
// This matches the level argument for ots::OTSContext::Message.
type MessageLevel int

const (
	MessageLevelError MessageLevel = iota
	MessageLevelWarning
)

func (m MessageLevel) String() string {
	switch m {
	case MessageLevelError:
		return "error"
	case MessageLevelWarning:
		return "warning"
	default:
		return ""
	}
}

// MessageFunc is called with a message.
type MessageFunc func(level MessageLevel, msg string)

// WithMessages specifies a handler for messages from OTS.
func WithMessages(fn MessageFunc) Option {
	return func(is *inst) {
		if fn != nil {
			is.message = fn
		}
	}
}

// WithMaxSize sets the maximum output length. By default, this is 8 times the
// input length.
func WithMaxSize(size int) Option {
	return func(is *inst) {
		if size != 0 {
			is.maxSize = size
		}
	}
}

// ErrSanitize is returned when font sanitization fails.
var ErrSanitize = errors.New("failed to sanitize font")

// Process sanitizes and re-serializes a TTF/OTF font (decompressing WOFF/WOFF2
// if required) into w with the specified options.
func Process(input []byte, opt ...Option) ([]byte, error) {
	Compile()

	is := &inst{
		index:       0xFFFFFFFF,
		tableAction: func(table Tag) TableAction { return TableActionDefault },
		message:     func(level MessageLevel, msg string) {},
		maxSize:     len(input) * 8,
	}
	for _, opt := range opt {
		if opt != nil {
			opt(is)
		}
	}

	id := instID.Add(1)
	insts.Store(id, is)
	defer insts.Delete(id)

	var err error
	ctx := context.Background()

	is.module, err = runtime.InstantiateModule(ctx, module, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		return nil, err
	}
	defer is.module.Close(ctx)

	var pInput uint32
	if r, err := is.module.ExportedFunction("gots_malloc").Call(ctx, uint64(len(input))); err != nil {
		return nil, err
	} else if len(r) != 1 {
		panic("wtf")
	} else {
		pInput = uint32(r[0])
	}
	if !is.module.Memory().Write(pInput, input) {
		panic("wtf")
	}

	var pOutputSize uint32
	if r, err := is.module.ExportedFunction("gots_malloc").Call(ctx, 4); err != nil {
		return nil, err
	} else if len(r) != 1 {
		panic("wtf")
	} else {
		pOutputSize = uint32(r[0])
	}
	if !is.module.Memory().WriteUint32Le(pOutputSize, uint32(is.maxSize)) {
		panic("wtf")
	}

	var pOutput uint32
	if r, err := is.module.ExportedFunction("gots_process").Call(ctx, uint64(id), uint64(pInput), uint64(len(input)), uint64(is.index), uint64(pOutputSize)); err != nil {
		return nil, err
	} else if len(r) != 1 {
		panic("wtf")
	} else {
		pOutput = uint32(r[0])
	}
	if pOutput == 0 {
		return nil, ErrSanitize
	}

	outputSize, ok := is.module.Memory().ReadUint32Le(pOutputSize)
	if !ok {
		panic("wtf")
	}

	output, ok := is.module.Memory().Read(pOutput, outputSize)
	if !ok {
		panic("wtf")
	}

	return append([]byte{}, output...), nil
}

// Extension is a helper to get the file extension for a font file (lowercase,
// including the leading dot).
//
// If no extension is known, an empty string is returned. Processed fonts will
// always return an extension since OTS has a whitelist of SFNT versions). The
// input file for fonts which have been processed successfully will also always
// return an extension.
func Extension(output []byte) string {
	switch {
	case bytes.HasPrefix(output, []byte{'O', 'T', 'T', 'O'}):
		return ".otf"
	case bytes.HasPrefix(output, []byte{'t', 'r', 'u', 'e'}):
		return ".ttf"
	case bytes.HasPrefix(output, []byte{0x00, 0x01, 0x00, 0x00}):
		return ".ttf"
	case bytes.HasPrefix(output, []byte{'t', 't', 'c', 'f'}):
		return ".ttc"
	case bytes.HasPrefix(output, []byte{'w', 'O', 'F', 'F'}): // input only, unpacked into one of the above
		return ".woff"
	case bytes.HasPrefix(output, []byte{'w', 'O', 'F', '2'}): // input only, unpacked into one of the above
		return ".woff2"
	default:
		return ""
	}
}

//go:generate wasm/build.sh
//go:embed wasm/ots.wasm
var wasm []byte

var (
	compile sync.Once
	runtime wazero.Runtime
	module  wazero.CompiledModule
	insts   sync.Map
	instID  atomic.Uint32
)

// Compile ensures the WebAssembly module is compiled, panicking if an error
// occurs. This is automatically called if it hasn't been already on the first
// call to Process.
func Compile() {
	compile.Do(func() {
		ctx := context.Background()
		runtime = wazero.NewRuntime(ctx)

		_, err := wasi_snapshot_preview1.Instantiate(ctx, runtime)
		if err != nil {
			panic(fmt.Errorf("gots: failed to instantiate wasi runtime: %w", err))
		}

		_, err = runtime.NewHostModuleBuilder("env").
			NewFunctionBuilder().WithFunc(gots_get_table_action).Export("gots_get_table_action").
			NewFunctionBuilder().WithFunc(gots_message).Export("gots_message").
			Instantiate(ctx)
		if err != nil {
			panic(fmt.Errorf("gots: failed to compile and instantiate host module: %w", err))
		}

		module, err = runtime.CompileModule(ctx, wasm)
		if err != nil {
			panic(fmt.Errorf("gots: failed to compile module: %w", err))
		}
	})
}

func gots_get_table_action(id uint32, tag uint32) uint32 {
	v, ok := insts.Load(id)
	if !ok {
		panic(fmt.Errorf("gots: id=%d doesn't exist", id))
	}
	return uint32(v.(*inst).tableAction([4]uint8{ // see OTS_UNTAG
		byte(tag >> 24),
		byte(tag >> 16),
		byte(tag >> 8),
		byte(tag >> 0),
	}))
}

func gots_message(id uint32, level uint32, message uint32, length uint32) {
	v, ok := insts.Load(id)
	if !ok {
		panic(fmt.Errorf("gots: id=%d doesn't exist", id))
	}
	msg, ok := v.(*inst).module.Memory().Read(message, length)
	if !ok {
		panic("wtf")
	}
	v.(*inst).message(MessageLevel(level), string(msg))
}
