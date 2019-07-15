#include "kythe/cxx/common/kzip_writer_aosp.h"

#include <openssl/sha.h>
#include <array>
#include <string>

#include "kythe/cxx/common/json_proto.h"
#include "absl/memory/memory.h"
#include "absl/strings/escaping.h"
#include "kythe/proto/analysis.pb.h"

namespace kythe {
namespace {

constexpr absl::string_view kRoot = "root/";
constexpr absl::string_view kUnitRoot = "root/units/";
constexpr absl::string_view kFileRoot = "root/files/";

}

std::string KzipWriter::SHA256Digest(absl::string_view content) {
  std::array<unsigned char, SHA256_DIGEST_LENGTH> buf;
  ::SHA256(reinterpret_cast<const unsigned char*>(content.data()),
           content.size(), buf.data());
  return absl::BytesToHexString(
      absl::string_view(reinterpret_cast<const char*>(buf.data()), buf.size()));
}

Status KzipWriter::WriteTextFile(const std::string& path,
                     absl::string_view content) {
  int32_t rc = zip_writer_.StartEntry(path.c_str(), ZipWriter::kCompress);
  if (rc == 0) {
    rc = zip_writer_.WriteBytes(content.data(), content.size());
  }
  if (rc == 0) {
    rc = zip_writer_.FinishEntry();
  }
  return rc ? InternalError(ZipWriter::ErrorCodeString(rc)) : OkStatus();
}

// Creates entries for the three directories if not already present.
int32_t KzipWriter::InitializeArchive() {
  if (initialized_) {
    return 0;
  }
  initialized_ = true;
  for (const auto name : {kRoot, kUnitRoot, kFileRoot}) {
    int32_t rc = zip_writer_.StartEntry(name.data(), 0);
    if (rc == 0) {
      rc = zip_writer_.FinishEntry();
    }
    if (rc) {
      return rc;
    }
  }
  return 0;
}


/* static */
StatusOr<IndexWriter> KzipWriter::Create(absl::string_view path) {
  FILE *fp = fopen(path.data(), "wb");
  if (!fp) {
    return UnimplementedError(strerror(errno));
  }
  return IndexWriter(absl::WrapUnique(new KzipWriter(fp)));
}

KzipWriter::KzipWriter(FILE *fp):fp_(fp), zip_writer_(fp), initialized_(false) {}

KzipWriter::~KzipWriter() {
  DCHECK(fp_ == nullptr) << "KzipWriterAosp::Close was not called!";
}

StatusOr<std::string> KzipWriter::WriteUnit(
    const kythe::proto::IndexedCompilation& unit) {
  int32_t rc = InitializeArchive();
  if (rc) {
      return InternalError(ZipWriter::ErrorCodeString(rc));
  }
  if (auto json = WriteMessageAsJsonToString(unit)) {
    auto file = InsertFile(kUnitRoot, std::move(*json));
    if (file.inserted()) {
      auto status = WriteTextFile(file.path(), file.contents());
      if (!status.ok()) {
        contents_.erase(file.path());
        return status;
      }
    }
    return std::string(file.digest());
  } else {
    return json.status();
  }
}

StatusOr<std::string> KzipWriter::WriteFile(absl::string_view content) {
  int32_t rc = InitializeArchive();
  if (rc) {
      return InternalError(ZipWriter::ErrorCodeString(rc));
  }
  auto file = InsertFile(kFileRoot, content);
  if (file.inserted()) {
    auto status = WriteTextFile(file.path(), file.contents());
    if (!status.ok()) {
      contents_.erase(file.path());
      return status;
    }
  }
  return std::string(file.digest());
}

Status KzipWriter::Close() {
  int32_t rc = zip_writer_.Finish();
  fclose(fp_);
  fp_ = nullptr;
  return rc ? InternalError(ZipWriter::ErrorCodeString(rc)) : OkStatus();
}

auto KzipWriter::InsertFile(absl::string_view root, absl::string_view content)
    -> InsertionResult {
  auto digest = SHA256Digest(content);
  auto path = absl::StrCat(root, digest);
  // Initially insert an empty string for the file content.
  auto result = InsertionResult{contents_.emplace(path, "")};
  if (result.inserted()) {
    // Only copy in the real content if it was actually inserted into the map.
    result.insertion.first->second = std::string(content);
  }
  return result;
}

inline absl::string_view KzipWriter::InsertionResult::digest() const {
  auto pos = path().find_last_of('/');
  if (pos == absl::string_view::npos) {
    return path();
  }
  return absl::ClippedSubstr(path(), pos + 1);

}

}  // namespace kythe
