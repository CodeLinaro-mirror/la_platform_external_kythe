#include "third_party/llvm/src/clang_builtin_headers.h"
// ANDROID_BUILD {
#include "clang_builtin_headers_resources.inc"
// }

const struct FileToc *builtin_headers_create() { return kPackedFiles; }
