#ifndef KYTHE_CXX_COMMON_KZIP_WRITER_AOSP_H_
#define KYTHE_CXX_COMMON_KZIP_WRITER_AOSP_H_

#include <unordered_set>

#include "absl/status/status.h"
#include "absl/status/statusor.h"
#include "absl/strings/string_view.h"
#include "kythe/cxx/common/index_writer.h"
#include "kythe/cxx/common/kzip_encoding.h"
#include "kythe/proto/analysis.pb.h"
#include "ziparchive/zip_writer.h"

namespace kythe {

/// \brief Kzip implementation of IndexWriter for AOSP.
/// see https://www.kythe.io/docs/kythe-kzip.html for format description.
class KzipWriter : public IndexWriterInterface {
 public:
  /// \brief Constructs a Kzip IndexWriter which will create and write to
  /// \param path Path to the file to create. Must not currently exist.
  static absl::StatusOr<IndexWriter> Create(
      absl::string_view path, KzipEncoding encoding = DefaultEncoding());

  /// \brief Destroys the KzipWriter.
  ~KzipWriter() override;

  /// \brief Writes the unit to the kzip file, returning its digest.
  absl::StatusOr<std::string> WriteUnit(
      const kythe::proto::IndexedCompilation& unit) override;

  /// \brief Writes the file contents to the kzip file, returning their digest.
  absl::StatusOr<std::string> WriteFile(absl::string_view content) override;

  /// \brief Flushes accumulated writes and closes the kzip file.
  /// Close must be called before the KzipWriter is destroyed!
  absl::Status Close() override;

 private:
  explicit KzipWriter(FILE* fp, KzipEncoding encoding);

  // Adds an entry <dir_name>/<content_digest> to the kzip file unless it
  // already exists. Returns <content_digest> on success.
  absl::StatusOr<std::string> InsertFile(absl::string_view dir_name,
                                         absl::string_view content_digest,
                                         absl::string_view content);
  absl::Status WriteTextFile(absl::string_view name, absl::string_view content);
  int32_t InitializeArchive();
  int32_t CreateDirEntry(absl::string_view dir_name);
  bool HasEncoding(KzipEncoding encoding);
  static KzipEncoding DefaultEncoding();

  FILE* fp_;
  ZipWriter zip_writer_;
  bool initialized_;  // Whether or not the `root` entry exists.
  KzipEncoding encoding_;
  std::unordered_set<std::string> contents_;
};

}  // namespace kythe
#endif  // KYTHE_CXX_COMMON_KZIP_WRITER_AOSP_H_
