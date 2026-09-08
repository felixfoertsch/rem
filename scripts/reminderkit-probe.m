// Developer-only probe. No store, permission request, or personal data.
#import <Foundation/Foundation.h>
#import <objc/runtime.h>
#import <objc/message.h>
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>

@interface REMProbeReminder : NSObject
@property(nonatomic, strong) NSSet *assignments;
- (id)storage;
@end
@implementation REMProbeReminder
- (id)storage { return self; }
@end
static id call0(id o, NSString *name) {
    SEL s = NSSelectorFromString(name);
    return [o respondsToSelector:s] ? ((id(*)(id,SEL))objc_msgSend)(o,s) : nil;
}
int main(void) {
    @autoreleasepool {
        dlopen("/System/Library/PrivateFrameworks/ReminderKit.framework/ReminderKit", RTLD_NOW | RTLD_LOCAL);
        dlopen("/System/Library/PrivateFrameworks/ReminderKitInternal.framework/ReminderKitInternal", RTLD_NOW | RTLD_LOCAL);
        for (NSString *name in @[@"REMListStorage", @"REMMembership"]) {
            Class c = NSClassFromString(name);
            printf("CLASS %s\n", name.UTF8String);
            unsigned int n = 0;
            Method *m = class_copyMethodList(c, &n);
            for (unsigned int i = 0; i < n; i++) printf("- %s %s\n", sel_getName(method_getName(m[i])), method_getTypeEncoding(m[i]));
            free(m);
            objc_property_t *p = class_copyPropertyList(c, &n);
            for (unsigned int i = 0; i < n; i++) printf("PROPERTY %s %s\n", property_getName(p[i]), property_getAttributes(p[i]));
            free(p);
        }
        // Find narrowly named identity/membership helpers, including categories.
        unsigned int n = 0;
        Class *classes = objc_copyClassList(&n);
        for (unsigned int i = 0; i < n; i++) {
            if (![NSStringFromClass(classes[i]) hasPrefix:@"REM"]) continue;
            unsigned int count = 0;
            Method *methods = class_copyMethodList(classes[i], &count);
            for (unsigned int j = 0; j < count; j++) {
                NSString *s = NSStringFromSelector(method_getName(methods[j]));
                if ([s localizedCaseInsensitiveContainsString:@"membershipsOfReminders"] ||
                    [s localizedCaseInsensitiveContainsString:@"currentUserShare"] ||
                    [s localizedCaseInsensitiveContainsString:@"shareParticipantID"] ||
                    [s localizedCaseInsensitiveContainsString:@"sectionForReminder"]) {
                    printf("HELPER %s %s %s\n", class_getName(classes[i]), s.UTF8String, method_getTypeEncoding(methods[j]));
                }
            }
            free(methods);
        }
        free(classes);
        @try {
            Class aClass = NSClassFromString(@"REMAssignment");
            SEL init = NSSelectorFromString(@"initWithObjectID:accountID:reminderID:assigneeID:originatorID:status:");
            id aid = call0(aClass,@"newObjectID");
            id accountID = call0(NSClassFromString(@"REMAccount"),@"newObjectID");
            id reminderID = call0(NSClassFromString(@"REMReminder"),@"newObjectID");
            id shareeID = call0(NSClassFromString(@"REMSharee"),@"newObjectID");
            REMProbeReminder *r = [REMProbeReminder new];
            id ctx = [NSClassFromString(@"REMReminderAssignmentContext") alloc];
            SEL initCtx = NSSelectorFromString(@"initWithReminder:");
            if ([ctx respondsToSelector:initCtx] && [aClass instancesRespondToSelector:init]) {
                ctx = ((id(*)(id,SEL,id))objc_msgSend)(ctx,initCtx,r);
                for (NSInteger status = 0; status < 4; status++) {
                    id a = ((id(*)(id,SEL,id,id,id,id,id,NSInteger))objc_msgSend)([aClass alloc],init,aid,accountID,reminderID,shareeID,shareeID,status);
                    r.assignments = a ? [NSSet setWithObject:a] : [NSSet set];
                    printf("ASSIGNMENT_STATUS %ld current=%s\n", (long)status, call0(ctx,@"currentAssignment") ? "YES" : "NO");
                }
            }
        } @catch (NSException *e) { printf("SYNTHETIC_PROBE_UNAVAILABLE %s\n",e.reason.UTF8String); }
    }
    return 0;
}
