// Native boundary tests: synthetic objects only; no store or user data.
// Compile on macOS with Foundation and EventKit. Including the bridge lets us
// exercise its real guards, not a second test-only implementation of them.
#import "../internal/reminderkit/bridge_darwin.m"
#include <stdio.h>

@interface RemTestReminder : NSObject
@property(nonatomic, copy) NSString *calendarItemIdentifier;
@end
@implementation RemTestReminder
@end

@interface RemTestMethods : NSObject
- (id)echo:(id)value;
- (id)number:(NSInteger)value;
- (double)wrongReturn;
- (BOOL)saveSynchronouslyWithError:(NSError **)error;
@end
@implementation RemTestMethods
- (id)echo:(id)value { return value; }
- (id)number:(NSInteger)value { return @(value); }
- (double)wrongReturn { return 1.0; }
- (BOOL)saveSynchronouslyWithError:(NSError **)error {
    *error = [NSError errorWithDomain:@"rem.test" code:1 userInfo:@{NSLocalizedDescriptionKey:@"synthetic save failure"}];
    return NO;
}
@end

static NSUInteger checks;
static void check(BOOL ok, NSString *name) {
    checks++;
    if (!ok) fail([@"TEST FAILED: " stringByAppendingString:name]);
}
static void rejects(void (^operation)(void), NSString *name) {
    BOOL rejected = NO;
    @try { operation(); } @catch (NSException *e) { rejected = YES; }
    check(rejected,name);
}
int main(void) {
    @autoreleasepool {
        @try {
            RemTestMethods *methods = [RemTestMethods new];
            check([invoke(methods,@"echo:",@[@"text"]) isEqual:@"text"],@"object argument/return ABI");
            check([invoke(methods,@"number:",@[@7]) isEqual:@7],@"NSInteger ABI");
            rejects(^{ invoke(methods,@"absent",@[]); },@"missing selector guard");
            rejects(^{ invoke(methods,@"echo:",@[]); },@"arity guard");
            rejects(^{ invoke(methods,@"number:",@[@"wrong"]); },@"argument type guard");
            rejects(^{ invoke(methods,@"wrongReturn",@[]); },@"return type guard");
            BOOL saveError = NO;
            @try { save(methods); } @catch (NSException *e) { saveError = [e.reason isEqualToString:@"synthetic save failure"]; }
            check(saveError,@"native save NSError propagation");

            RemTestReminder *one = [RemTestReminder new], *two = [RemTestReminder new];
            one.calendarItemIdentifier = @"ABC-1"; two.calendarItemIdentifier = @"ABC-2";
            NSArray *reminders = (id)@[one,two];
            check([(id)findReminder(reminders,@"abc-1") isEqual:reminders[0]],@"exact ID priority");
            check([(id)findReminder(reminders,@"x-apple-reminder://ABC-2") isEqual:reminders[1]],@"URL ID normalization");
            rejects(^{ findReminder(reminders,@"ABC"); },@"ambiguous prefix rejected");
            rejects(^{ findReminder(reminders,@""); },@"empty prefix rejected");
            rejects(^{ findReminder(reminders,@"missing"); },@"missing reminder rejected");

            NSUUID *ownerID = [NSUUID UUID], *otherID = [NSUUID UUID];
            NSDictionary *owner = @{@"objectID":ownerID,@"displayName":@"Owner",@"address":@"owner@example.com",@"accessLevel":@0};
            NSDictionary *other = @{@"objectID":otherID,@"displayName":@"Other",@"address":@"other@example.com",@"accessLevel":@0};
            NSDictionary *list = @{@"isShared":@YES,@"isOwnedByMe":@YES,@"sharedOwnerID":ownerID,
                @"shareeContext":@{@"sharees":@[owner,other],@"sharedOwner":owner}};
            NSArray *roster = participants(list);
            check(roster.count == 2,@"owner deduplicated");
            NSUInteger meCount = 0;
            for (NSDictionary *person in roster) {
                if ([person[@"is_me"] boolValue]) meCount++;
                check(publicParticipant(person)[@"_object_id"] == nil,@"private object ID removed from JSON");
            }
            check(meCount == 1,@"native owner identity recognized");
            check(participants(@{@"isShared":@NO}).count == 0,@"unshared list has no participants");
            NSDictionary *orphan = @{@"assignmentContext":@{@"currentAssignment":@{@"assigneeID":[NSUUID UUID]}}};
            NSDictionary *meta = metadata(orphan,list);
            check([meta[@"assignment_available"] boolValue] && meta[@"assigned_to"] != NSNull.null,@"orphan assignment ID preserved");
            NSDictionary *unassigned = metadata(@{@"assignmentContext":@{}},list);
            check([unassigned[@"assignment_available"] boolValue] && unassigned[@"assigned_to"] == NSNull.null,@"unassigned is known null");
            check(![metadata(@{},list)[@"assignment_available"] boolValue],@"unavailable is not unassigned");

            NSDictionary *diagnostics = handle(@{@"op":@"diagnostics"});
            check([diagnostics[@"permission_requested"] isEqual:@NO],@"diagnostics requests no permission");
            // Exercise the production initializer invocation with real native
            // objects. No claims are made about assignment status semantics.
            Class cls = NSClassFromString(@"REMAssignment");
            if (cls) {
                id aid = invoke(cls,@"newObjectID",@[]);
                id account = invoke(NSClassFromString(@"REMAccount"),@"newObjectID",@[]);
                id reminderID = invoke(NSClassFromString(@"REMReminder"),@"newObjectID",@[]);
                id sharee = invoke(NSClassFromString(@"REMSharee"),@"newObjectID",@[]);
                id assignment = newObject(@"REMAssignment",@"initWithObjectID:accountID:reminderID:assigneeID:originatorID:status:",@[aid,account,reminderID,sharee,sharee,@0]);
                check([get(assignment,@"assigneeID") isEqual:sharee],@"native constructor argument ABI");
                id storage = [NSClassFromString(@"REMReminderStorage") new];
                [storage setValue:[NSSet setWithObject:assignment] forKey:@"assignments"];
                check([get(get(storage,@"currentAssignment"),@"assigneeID") isEqual:sharee],@"native storage can read synthetic assignment");
            }
            printf("PASS: %lu native boundary checks; no live Reminders access\n",(unsigned long)checks);
        } @catch (NSException *e) {
            fprintf(stderr,"%s\n",e.reason.UTF8String);
            return 1;
        }
    }
    return 0;
}
