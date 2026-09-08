// Developer-only in-memory semantics probe. Never opens a store.
#import <Foundation/Foundation.h>
#import <objc/runtime.h>
#import <objc/message.h>
#include <dlfcn.h>
#include <stdio.h>
static id call0(id o, NSString *name) {
    SEL s = NSSelectorFromString(name);
    return [o respondsToSelector:s] ? ((id(*)(id,SEL))objc_msgSend)(o,s) : nil;
}
int main(void) {
    @autoreleasepool {
        dlopen("/System/Library/PrivateFrameworks/ReminderKit.framework/ReminderKit", RTLD_NOW | RTLD_LOCAL);
        @try {
            Class ac = NSClassFromString(@"REMAssignment");
            SEL init = NSSelectorFromString(@"initWithObjectID:accountID:reminderID:assigneeID:originatorID:status:");
            id aid = call0(ac,@"newObjectID");
            id account = call0(NSClassFromString(@"REMAccount"),@"newObjectID");
            id reminder = call0(NSClassFromString(@"REMReminder"),@"newObjectID");
            id sharee = call0(NSClassFromString(@"REMSharee"),@"newObjectID");
            id storage = [NSClassFromString(@"REMReminderStorage") new];
            for (NSInteger status = 0; status < 4; status++) {
                id a = ((id(*)(id,SEL,id,id,id,id,id,NSInteger))objc_msgSend)([ac alloc],init,aid,account,reminder,sharee,sharee,status);
                [storage setValue:[NSSet setWithObject:a] forKey:@"assignments"];
                printf("ASSIGNMENT_STATUS %ld current=%s description=%s\n", (long)status, call0(storage,@"currentAssignment") ? "YES" : "NO", [[a description] UTF8String]);
            }
            unsigned int n = 0;
            Method *m = class_copyMethodList(NSClassFromString(@"REMReminderStorage"), &n);
            for (unsigned int i=0;i<n;i++) {
                NSString *s = NSStringFromSelector(method_getName(m[i]));
                if ([s localizedCaseInsensitiveContainsString:@"assign"] || [s localizedCaseInsensitiveContainsString:@"section"]) printf("STORAGE %s %s\n",s.UTF8String,method_getTypeEncoding(m[i]));
            }
            free(m);
        } @catch (NSException *e) { printf("PROBE_ERROR %s\n",e.reason.UTF8String); return 1; }
    }
    return 0;
}
