// Developer-only capability probe. Does not open a store or read user data.
#import <Foundation/Foundation.h>
#import <objc/runtime.h>
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>

int main(void) {
    @autoreleasepool {
        dlopen("/System/Library/PrivateFrameworks/ReminderKit.framework/ReminderKit", RTLD_NOW | RTLD_LOCAL);
        dlopen("/System/Library/PrivateFrameworks/ReminderKitInternal.framework/ReminderKitInternal", RTLD_NOW | RTLD_LOCAL);
        unsigned int count = 0;
        Class *classes = objc_copyClassList(&count);
        NSMutableArray<NSString *> *names = [NSMutableArray array];
        for (unsigned int i = 0; i < count; i++) {
            NSString *name = NSStringFromClass(classes[i]);
            if (![name hasPrefix:@"REM"]) continue;
            if ([name containsString:@"Assignment"] || [name containsString:@"Section"] ||
                [name isEqualToString:@"REMSharee"] || [name isEqualToString:@"REMList"] ||
                [name isEqualToString:@"REMSaveRequest"] || [name isEqualToString:@"REMObjectID"] ||
                [name isEqualToString:@"REMReminder"]) {
                [names addObject:name];
            }
        }
        free(classes);
        [names sortUsingSelector:@selector(compare:)];
        for (NSString *name in names) {
            printf("\nCLASS %s\n", name.UTF8String);
            Class cls = NSClassFromString(name);
            for (int meta = 0; meta < 2; meta++) {
                unsigned int n = 0;
                Method *methods = class_copyMethodList(meta ? object_getClass(cls) : cls, &n);
                for (unsigned int i = 0; i < n; i++) {
                    printf("%c %s %s\n", meta ? '+' : '-', sel_getName(method_getName(methods[i])), method_getTypeEncoding(methods[i]));
                }
                free(methods);
            }
            unsigned int n = 0;
            objc_property_t *properties = class_copyPropertyList(cls, &n);
            for (unsigned int i = 0; i < n; i++) {
                printf("PROPERTY %s %s\n", property_getName(properties[i]), property_getAttributes(properties[i]));
            }
            free(properties);
        }
    }
    return 0;
}
