//go:build darwin

#import <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>

#include "scrollbars_darwin.h"

// The global-domain key the Appearance pane writes. NSUserDefaults searches
// the global domain after the application's own, so the standard defaults
// answer it without naming a domain, and an application that has written the
// key for itself overrides the user's choice the way the platform lets it.
static NSString *const vgio_desktop_scroll_bars_key = @"AppleShowScrollBars";

char *vgio_desktop_scroll_bars(void) {
	@autoreleasepool {
		NSString *value = [[NSUserDefaults standardUserDefaults]
			stringForKey:vgio_desktop_scroll_bars_key];
		if (value == nil) {
			return NULL;
		}
		const char *utf8 = [value UTF8String];
		if (utf8 == NULL) {
			return NULL;
		}
		return strdup(utf8);
	}
}
