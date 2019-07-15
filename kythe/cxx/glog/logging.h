/*
 * Android-compatible logging.h to avoid pulling in Google logging package.
 */

#if !defined(GLOG_LOGGING_H_)
#define GLOG_LOGGING_H_
#include "android-base/logging.h"
#define DFATAL FATAL
#define VLOG(verbose_level) LOG(VERBOSE)

namespace google {
void InitGoogleLogging(const char *argv0);
}

#endif // GLOG_LOGGING_H
