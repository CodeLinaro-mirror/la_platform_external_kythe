#ifndef KYTHE_CXX_COMMON_KZIP_WRITER_AOSP_H_
#define KYTHE_CXX_COMMON_KZIP_WRITER_AOSP_H_

#include <unordered_map>

#include "absl/strings/string_view.h"
#include "kythe/cxx/common/index_writer.h"
#include "kythe/cxx/common/status_or.h"
#include "kythe/proto/analysis.pb.h"
#include "ziparchive/zip_writer.h"

namespace kythe {

/// \brief Kzip implementation of IndexWriter for AOSP.
/// see https://www.kythe.io/docs/kythe-kzip.html for format description.
class KzipWriter : public IndexWriterInterface {
 public:
  /// \brief Constructs a Kzip IndexWriter which will create and write to
  /// \param path Path to the file to create. Must not currently exist.
  static StatusOr<IndexWriter> Create(absl::string_view path);

  /// \brief Destroys the KzipWriter.
  ~KzipWriter() override;

  /// \brief Writes the unit to the kzip file, returning its digest.
  StatusOr<std::string> WriteUnit(
      const kythe::proto::IndexedCompilation& unit) override;

  /// \brief Writes the file contents to the kzip file, returning their digest.
  StatusOr<std::string> WriteFile(absl::string_view content) override;

  /// \brief Flushes accumulated writes and closes the kzip file.
  /// Close must be called before the KzipWriter is destroyed!
  Status Close() override;

 private:
  using Path = std::string;
  using Contents = std::string;
  using FileMap = std::unordered_map<Path, Contents>;

  struct InsertionResult {
    absl::string_view digest() const;
    const std::string& path() const { return insertion.first->first; }
    absl::string_view contents() const { return insertion.first->second; }
    bool inserted() const { return insertion.second; }

    std::pair<FileMap::iterator, bool> insertion;
  };

  explicit KzipWriter(FILE *fp);

  InsertionResult InsertFile(absl::string_view root, absl::string_view content);
  Status WriteTextFile(const std::string& path, absl::string_view content);
  int32_t InitializeArchive();
  static std::string SHA256Digest(absl::string_view content);

  FILE *fp_;
  ZipWriter zip_writer_;
  bool initialized_ = false;  // Whether or not the `root` entry exists.
  FileMap contents_;
};

}  // namespace kythe
#endif  // KYTHE_CXX_COMMON_KZIP_WRITER_AOSP_H_
