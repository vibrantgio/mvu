// download_darwin.m is the platform half of the cloud download: the ask
// that starts one, and the read that says how far it has got. Nothing here
// keeps state — the item's state is the platform's, read back off the item
// every time it is wanted.

#import <Foundation/Foundation.h>

#include <stdlib.h>
#include <string.h>

#include "download_darwin.h"

// vgio_download_copy hands a string across the boundary as a buffer Go can
// read after the pool has drained. nil answers NULL, which is how both
// calls say "no error".
static char *vgio_download_copy(NSString *s) {
	if (s == nil) {
		return NULL;
	}
	return strdup([s UTF8String]);
}

char *vgio_download_start(const char *path) {
	@autoreleasepool {
		NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
		NSError *err = nil;
		if ([[NSFileManager defaultManager] startDownloadingUbiquitousItemAtURL:url error:&err]) {
			return NULL;
		}
		if (err == nil) {
			return strdup("the platform refused to start the download");
		}
		return vgio_download_copy([err localizedDescription]);
	}
}

int vgio_download_state(const char *path, char **failure) {
	@autoreleasepool {
		*failure = NULL;
		NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
		NSDictionary *values = [url resourceValuesForKeys:@[
			NSURLUbiquitousItemDownloadingStatusKey,
			NSURLUbiquitousItemDownloadingErrorKey,
		] error:NULL];
		NSError *recorded = values[NSURLUbiquitousItemDownloadingErrorKey];
		if (recorded != nil) {
			*failure = vgio_download_copy([recorded localizedDescription]);
		}
		NSString *status = values[NSURLUbiquitousItemDownloadingStatusKey];
		if ([status isEqualToString:NSURLUbiquitousItemDownloadingStatusNotDownloaded]) {
			return VGIO_DOWNLOAD_NOT_DOWNLOADED;
		}
		if ([status isEqualToString:NSURLUbiquitousItemDownloadingStatusDownloaded]) {
			return VGIO_DOWNLOAD_DOWNLOADED;
		}
		if ([status isEqualToString:NSURLUbiquitousItemDownloadingStatusCurrent]) {
			return VGIO_DOWNLOAD_CURRENT;
		}
		return VGIO_DOWNLOAD_UNKNOWN;
	}
}
