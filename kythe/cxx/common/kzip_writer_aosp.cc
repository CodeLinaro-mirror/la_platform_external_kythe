#include "kythe/cxx/common/kzip_writer_aosp.h"

#include <openssl/sha.h>

#include <array>
#include <string>

#include "absl/memory/memory.h"
#include "absl/strings/escaping.h"
#include "kythe/cxx/common/json_proto.h"
#include "kythe/proto/analysis.pb.h"

namespace kythe {
namespace {

constexpr absl::string_view kRoot = "root/";
constexpr absl::string_view kJsonUnitRoot = "root/units/";
constexpr absl::string_view kProtoUnitRoot = "root/pbunits/";
constexpr absl::string_view kFileRoot = "root/files/";
// Set all file modified times to 0 so zip file diffs only show content diffs,
// not zip creation time diffs.
constexpr time_t kModTime = 0;

std::string SHA256Digest(absl::string_view content) {
  std::array<unsigned char, SHA256_DIGEST_LENGTH> buf;
  ::SHA256(reinterpret_cast<const unsigned char*>(content.data()),
           content.size(), buf.data());
  return absl::BytesToHexString(
      absl::string_view(reinterpret_cast<const char*>(buf.data()), buf.size()));
}

}  // namespace

bool KzipWriter::HasEncoding(KzipEncoding encoding) {
  return encoding_ == encoding || encoding_ == KzipEncoding::kAll;
}

absl::Status KzipWriter::WriteTextFile(absl::string_view path,
                                 absl::string_view content) {
  int32_t rc = zip_writer_.StartEntryWithTime(path.data(), ZipWriter::kCompress,
                                              kModTime);
  if (rc == 0) {
    rc = zip_writer_.WriteBytes(content.data(), content.size());
  }
  if (rc == 0) {
    rc = zip_writer_.FinishEntry();
  }
  return rc ? absl::InternalError(ZipWriter::ErrorCodeString(rc)) : absl::OkStatus();
}

int32_t KzipWriter::CreateDirEntry(absl::string_view path) {
  auto rc = zip_writer_.StartEntryWithTime(path.data(), 0, kModTime);
  if (rc == 0) {
    rc = zip_writer_.FinishEntry();
  }
  return rc;
}

// Creates entries for the directories if not already present.
int32_t KzipWriter::InitializeArchive() {
  if (initialized_) {
    return 0;
  }
  initialized_ = true;
  auto rc = CreateDirEntry(kRoot);
  if (rc == 0) {
    rc = CreateDirEntry(kFileRoot);
  }
  if (rc == 0 && HasEncoding(KzipEncoding::kJson)) {
    rc = CreateDirEntry(kJsonUnitRoot);
  }
  if (rc == 0 && HasEncoding(KzipEncoding::kProto)) {
    rc = CreateDirEntry(kProtoUnitRoot);
  }
  return rc;
}

/* static */
StatusOr<IndexWriter> KzipWriter::Create(absl::string_view path,
                                         KzipEncoding encoding) {
  FILE* fp = fopen(path.data(), "wb");
  if (!fp) {
    return absl::UnimplementedError(strerror(errno));
  }
  return IndexWriter(absl::WrapUnique(new KzipWriter(fp, encoding)));
}

KzipWriter::KzipWriter(FILE* fp, KzipEncoding encoding)
    : fp_(fp), zip_writer_(fp), initialized_(false), encoding_(encoding) {}

KzipWriter::~KzipWriter() {
  DCHECK(fp_ == nullptr) << "KzipWriterAosp::Close was not called!";
}

StatusOr<std::string> KzipWriter::WriteUnit(
    const kythe::proto::IndexedCompilation& unit) {
  int32_t rc = InitializeArchive();
  if (rc) {
    return absl::InternalError(ZipWriter::ErrorCodeString(rc));
  }
  auto json = WriteMessageAsJsonToString(unit);
  if (!json) {
    return json.status();
  }

  auto digest = SHA256Digest(*json);
  StatusOr<std::string> result = absl::InternalError("unsupported encoding");
  if (HasEncoding(KzipEncoding::kJson)) {
    if (!(result = InsertFile(kJsonUnitRoot, digest, *json))) {
      return result;
    }
  }
  if (HasEncoding(KzipEncoding::kProto)) {
    std::string contents;
    if (!unit.SerializeToString(&contents)) {
      return absl::InternalError("Failure serializing compilation unit");
    }
    result = InsertFile(kProtoUnitRoot, digest, contents);
  }
  return result;
}

StatusOr<std::string> KzipWriter::WriteFile(absl::string_view content) {
  int32_t rc = InitializeArchive();
  if (rc) {
    return absl::InternalError(ZipWriter::ErrorCodeString(rc));
  }
  return InsertFile(kFileRoot, SHA256Digest(content), content);
}

absl::Status KzipWriter::Close() {
  int32_t rc = zip_writer_.Finish();
  fclose(fp_);
  fp_ = nullptr;
  return rc ? absl::InternalError(ZipWriter::ErrorCodeString(rc)) : absl::OkStatus();
}

StatusOr<std::string> KzipWriter::InsertFile(absl::string_view dir,
                                             absl::string_view content_digest,
                                             absl::string_view content) {
  auto kzip_entry = absl::StrCat(dir, content_digest);
  // Keep track of the inserted entries, reject duplicates
  auto insertion = contents_.insert(kzip_entry);
  if (insertion.second) {
    auto status = WriteTextFile(kzip_entry, std::string(content));
    if (!status.ok()) {
      contents_.erase(insertion.first);
      return status;
    }
  }
  return std::string(content_digest);
}

KzipEncoding KzipWriter::DefaultEncoding() {
  const char* env_encoding = getenv("KYTHE_KZIP_ENCODING");
  if (env_encoding != nullptr && *env_encoding) {
    if (strcasecmp(env_encoding, "json") == 0) {
      return KzipEncoding::kJson;
    } else if (strcasecmp(env_encoding, "proto") == 0) {
      return KzipEncoding::kProto;
    } else if (strcasecmp(env_encoding, "all") == 0) {
      return KzipEncoding::kAll;
    } else {
      LOG(ERROR) << "Unknown encoding '" << env_encoding << "', using JSON";
    }
  }
  return KzipEncoding::kJson;
}

}  // namespace kythe
