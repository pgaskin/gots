#include <opentype-sanitiser.h>
#include <ots-memory-stream.h>
#include <cstdarg>
#include <cstdint>
#include <cstdio>

__attribute__((import_module("env"), import_name("gots_get_table_action")))
ots::TableAction gots_get_table_action(uint32_t id, uint32_t tag);

__attribute__((import_module("env"), import_name("gots_message")))
void gots_message(uint32_t id, int level, const char *message, size_t length);

class GOTSContext : public ots::OTSContext {
public:
    GOTSContext(uint32_t id) : id(id) {}
    virtual ~GOTSContext() {}

    virtual void Message(int level OTS_UNUSED, const char *format OTS_UNUSED, ...) MSGFUNC_FMT_ATTR {
        va_list va;
        va_start(va, format);
        char message[1024];
        const int len = std::vsnprintf(message, sizeof(message), format, va);
        if (len >= 0) gots_message(id, level, message, len);
        va_end(va);
    }

    virtual ots::TableAction GetTableAction(uint32_t tag OTS_UNUSED) {
        return gots_get_table_action(id, tag);
    }

private:
    uint32_t id;
};

__attribute__((visibility("default"), export_name("gots_malloc")))
void *gots_malloc(size_t length) {
    return operator new(length);
}

__attribute__((visibility("default"), export_name("gots_process")))
void *gots_process(uint32_t id, const uint8_t *input, size_t length, uint32_t index, size_t *outputSize) {
    auto stream = new ots::ExpandingMemoryStream(length, *outputSize);
    if (!GOTSContext(id).Process(stream, input, length, index)) return nullptr;
    *outputSize = stream->Tell();
    return stream->get();
}
