// Developer-only capability probe. No store, permissions, or user data.
#import <Foundation/Foundation.h>
#import <objc/runtime.h>
#import <objc/message.h>
#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>

@interface REMProbeReminder : NSObject
@property(nonatomic, strong) NSSet *assignments;
@end
@implementation REMProbeReminder
@end

static id call0(id target, NSString *name) {
    SEL s = NSSelectorFromString(name);
    return [target respondsToSelector:s] ? ((id(*)(id,SEL))objc_msgSend)(target,s) : nil;
}

int main(void) {
    @autoreleasepool {
        dlopen("/System/Library/PrivateFrameworks/ReminderKit.framework/ReminderKit", RTLD_NOW | RTLD_LOCAL);
        dlopen("/System/Library/PrivateFrameworks/ReminderKitInternal.framework/ReminderKitInternal", RTLD_NOW | RTLD_LOCAL);
        NSMutableSet *seen = [NSMutableSet set];
        for (NSString *name in @[@"REMListSectionContext", @"REMListSectionContextChangeItem", @"REMListShareeContext", @"REMMemberships", @"REMListChangeItem", @"REMReminderChangeItem", @"REMStore"]) {
            for (Class cls = NSClassFromString(name); cls && cls != [NSObject class]; cls = class_getSuperclass(cls)) {
                if ([seen containsObject:NSStringFromClass(cls)]) continue;
                [seen addObject:NSStringFromClass(cls)];
                printf("\nCLASS %s\n", class_getName(cls));
                for (int meta = 0; meta < 2; meta++) {
                    unsigned int n = 0;
                    Method *methods = class_copyMethodList(meta ? object_getClass(cls) : cls, &n);
                    for (unsigned int i = 0; i < n; i++) {
                        printf("%c %s %s\n", meta ? '+' : '-', sel_getName(method_getName(methods[i])), method_getTypeEncoding(methods[i]));
                    }
                    free(methods);
                }
                unsigned int n = 0;
                objc_property_t *props = class_copyPropertyList(cls, &n);
                for (unsigned int i = 0; i < n; i++) printf("PROPERTY %s %s\n", property_getName(props[i]), property_getAttributes(props[i]));
                free(props);
            }
        }
        // Determine which assignment status the native read context recognizes,
        // using only synthetic in-memory objects, never a user's reminder.
        @try {
            Class assignment = NSClassFromString(@"REMAssignment");
            SEL init = NSSelectorFromString(@"initWithObjectID:accountID:reminderID:assigneeID:originatorID:status:");
            id aid = call0(assignment, @"newObjectID");
            id accountID = call0(NSClassFromString(@"REMAccount"), @"newObjectID");
            id reminderID = call0(NSClassFromString(@"REMReminder"), @"newObjectID");
            id shareeID = call0(NSClassFromString(@"REMSharee"), @"newObjectID");
            REMProbeReminder *r = [REMProbeReminder new];
            id ctx = [NSClassFromString(@"REMReminderAssignmentContext") alloc];
            SEL initCtx = NSSelectorFromString(@"initWithReminder:");
            if ([ctx respondsToSelector:initCtx] && [assignment instancesRespondToSelector:init]) {
                ctx = ((id(*)(id,SEL,id))objc_msgSend)(ctx,initCtx,r);
                for (NSInteger status = 0; status < 4; status++) {
                    id a = ((id(*)(id,SEL,id,id,id,id,id,NSInteger))objc_msgSend)([assignment alloc],init,aid,accountID,reminderID,shareeID,shareeID,status);
                    r.assignments = a ? [NSSet setWithObject:a] : [NSSet set];
                    printf("ASSIGNMENT_STATUS %ld current=%s\n", (long)status, call0(ctx,@"currentAssignment") ? "YES" : "NO");
                }
            }
        } @catch (NSException *e) {
            printf("SYNTHETIC_PROBE_UNAVAILABLE %s\n", e.reason.UTF8String);
        }
    }
    return 0;
}
