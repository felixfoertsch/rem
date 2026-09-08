//go:build darwin && cgo

// A narrow, dynamically guarded extension to go-eventkit. No direct database writes,
// permission bypass, AppleScript, external helper process, or private headers.
#import <Foundation/Foundation.h>
#import <EventKit/EventKit.h>
#import <objc/runtime.h>
#import <objc/message.h>
#include <dlfcn.h>
#include <stdlib.h>
#include <string.h>

static void fail(NSString *message) {
    @throw [NSException exceptionWithName:@"RemCollaborationError" reason:message userInfo:nil];
}

// NSInvocation checks arity and argument/return types before crossing a private
// ABI. KVC below is intentional for ReminderKit's @dynamic storage properties.
static id invoke(id target, NSString *method, NSArray *args) {
    SEL selector = NSSelectorFromString(method);
    if (!target || ![target respondsToSelector:selector]) fail([@"Unavailable native method: " stringByAppendingString:method]);
    NSMethodSignature *signature = [target methodSignatureForSelector:selector];
    if (!signature || signature.numberOfArguments != args.count + 2) fail(@"Native method signature changed");
    NSInvocation *inv = [NSInvocation invocationWithMethodSignature:signature];
    inv.target = target;
    inv.selector = selector;
    for (NSUInteger i = 0; i < args.count; i++) {
        const char *type = [signature getArgumentTypeAtIndex:i + 2];
        id value = args[i] == NSNull.null ? nil : args[i];
        if (type[0] == '@') { [inv setArgument:&value atIndex:i + 2]; }
        else if (strcmp(type, @encode(NSInteger)) == 0 && [value isKindOfClass:NSNumber.class]) {
            NSInteger n = [value integerValue]; [inv setArgument:&n atIndex:i + 2];
        } else if (strcmp(type, @encode(BOOL)) == 0 && [value isKindOfClass:NSNumber.class]) {
            BOOL b = [value boolValue]; [inv setArgument:&b atIndex:i + 2];
        } else fail(@"Unsupported native argument type; no change was saved");
    }
    const char *ret = signature.methodReturnType;
    if (ret[0] != '@' && ret[0] != 'v') fail(@"Unsupported native return type");
    [inv invoke];
    if (ret[0] == 'v') return nil;
    __unsafe_unretained id value = nil;
    [inv getReturnValue:&value];
    return value;
}

static id get(id object, NSString *key) {
    if (!object) fail([@"Missing native object for " stringByAppendingString:key]);
    return [object valueForKey:key];
}
static id optional(id object, NSString *key) {
    @try { return object ? [object valueForKey:key] : nil; } @catch (NSException *e) { return nil; }
}
static NSString *text(id value) { return [value isKindOfClass:NSString.class] ? value : @""; }
static NSString *objectIDString(id oid) {
    if ([oid isKindOfClass:NSUUID.class]) return [oid UUIDString];
    return text(optional(optional(oid,@"uuid"),@"UUIDString"));
}
static BOOL equalText(NSString *a, NSString *b) {
    return a.length && b.length && [a caseInsensitiveCompare:b] == NSOrderedSame;
}

static id withError(id target, NSString *name, id arg) {
    SEL sel = NSSelectorFromString(name);
    NSMethodSignature *sig = [target methodSignatureForSelector:sel];
    if (![target respondsToSelector:sel] || !sig || sig.numberOfArguments != 4 ||
        sig.methodReturnType[0] != '@' || [sig getArgumentTypeAtIndex:2][0] != '@' ||
        strcmp([sig getArgumentTypeAtIndex:3], "^@") != 0) fail([@"Unavailable native fetch: " stringByAppendingString:name]);
    NSError *error = nil;
    id result = ((id(*)(id,SEL,id,NSError *__autoreleasing *))objc_msgSend)(target,sel,arg,&error);
    if (error) fail(error.localizedDescription);
    if (!result) fail([@"Native fetch returned no result: " stringByAppendingString:name]);
    return result;
}
static void save(id request) {
    SEL sel = NSSelectorFromString(@"saveSynchronouslyWithError:");
    NSMethodSignature *sig = [request methodSignatureForSelector:sel];
    if (![request respondsToSelector:sel] || !sig || sig.numberOfArguments != 3 ||
        strcmp(sig.methodReturnType,@encode(BOOL)) || strcmp([sig getArgumentTypeAtIndex:2],"^@")) fail(@"Native save API unavailable; nothing saved");
    NSError *error = nil;
    if (!((BOOL(*)(id,SEL,NSError *__autoreleasing *))objc_msgSend)(request,sel,&error))
        fail(error.localizedDescription ?: @"Native save failed");
}
static id newObject(NSString *className, NSString *initName, NSArray *args) {
    Class cls = NSClassFromString(className);
    if (!cls) fail([@"Unavailable native class: " stringByAppendingString:className]);
    id value = invoke([cls alloc],initName,args);
    if (!value) fail([@"Could not initialize " stringByAppendingString:className]);
    return value;
}
static id backingObject(id ekObject, NSString *expectedClass) {
    id backing = invoke(ekObject,@"backingObject",@[]);
    for (Class cls = object_getClass(backing); cls; cls = class_getSuperclass(cls)) {
        Ivar ivar = class_getInstanceVariable(cls,"_remObject");
        if (!ivar) continue;
        const char *type = ivar_getTypeEncoding(ivar);
        if (!type || type[0] != '@') fail(@"Native backing-object layout changed");
        id object = object_getIvar(backing,ivar);
        if ([object isKindOfClass:NSClassFromString(expectedClass)]) return object;
        break;
    }
    fail(@"This account or macOS version does not expose a supported ReminderKit object");
    return nil;
}
static NSArray<EKReminder *> *fetchReminders(EKEventStore *store, NSArray<EKCalendar *> *calendars) {
    dispatch_semaphore_t done = dispatch_semaphore_create(0);
    __block NSArray *found = nil;
    id token = [store fetchRemindersMatchingPredicate:[store predicateForRemindersInCalendars:calendars]
        completion:^(NSArray<EKReminder *> *items) { found = items; dispatch_semaphore_signal(done); }];
    if (dispatch_semaphore_wait(done,dispatch_time(DISPATCH_TIME_NOW,30*NSEC_PER_SEC))) {
        [store cancelFetchRequest:token]; fail(@"Reminders fetch timed out; no change was made");
    }
    if (!found) fail(@"Reminders fetch failed; check permission and account availability");
    return found;
}
static NSString *normalizeID(NSString *s) {
    NSString *prefix = @"x-apple-reminder://";
    return [[s lowercaseString] hasPrefix:prefix] ? [s substringFromIndex:prefix.length] : s;
}
static EKReminder *findReminder(NSArray<EKReminder *> *all, NSString *query) {
    query = normalizeID(query);
    if (!query.length) fail(@"A reminder ID is required");
    for (EKReminder *r in all) if (equalText(r.calendarItemIdentifier,query)) return r;
    EKReminder *match = nil;
    for (EKReminder *r in all) {
        if (![[r.calendarItemIdentifier lowercaseString] hasPrefix:query.lowercaseString]) continue;
        if (match) fail(@"Ambiguous reminder ID; use the full ID from rem list --output json");
        match = r;
    }
    if (!match) fail(@"Reminder not found");
    return match;
}
static EKCalendar *findList(EKEventStore *store, NSString *query) {
    NSArray *all = [store calendarsForEntityType:EKEntityTypeReminder];
    for (EKCalendar *c in all) if (equalText(c.calendarIdentifier,query)) return c;
    EKCalendar *match = nil;
    for (EKCalendar *c in all) {
        if (!equalText(c.title,query)) continue;
        if (match) fail(@"Ambiguous list name; use its exact ID from rem lists --output json");
        match = c;
    }
    if (!match) fail(@"List not found");
    return match;
}

// Each record keeps the real REMObjectID in process. It is removed before JSON
// serialization; strings/emails are never passed to native assignment methods.
static NSArray<NSDictionary *> *participants(id list) {
    if (![get(list,@"isShared") boolValue]) return @[];
    id context = get(list,@"shareeContext");
    NSArray *sharees = get(context,@"sharees");
    if (![sharees isKindOfClass:NSArray.class]) fail(@"Native participant roster is unavailable");
    NSMutableArray *source = [sharees mutableCopy];
    id owner = optional(context,@"sharedOwner");
    if (owner) [source addObject:owner];
    id ownerID = optional(list,@"sharedOwnerID");
    NSString *ownerString = objectIDString(ownerID);
    NSString *current = text(optional(list,@"currentUserShareParticipantID"));
    BOOL owned = [get(list,@"isOwnedByMe") boolValue];
    NSMutableDictionary *records = [NSMutableDictionary dictionary];
    for (id sharee in source) {
        id oid = get(sharee,@"objectID");
        NSString *identifier = objectIDString(oid);
        if (!identifier.length) fail(@"A participant has no usable native ID");
        BOOL me = equalText(identifier,current) || equalText(text(optional(oid,@"stringRepresentation")),current) ||
            (owned && equalText(identifier,ownerString));
        records[identifier] = [@{@"id":identifier,@"name":text(optional(sharee,@"displayName")),
            @"address":text(optional(sharee,@"address")),@"access_level":get(sharee,@"accessLevel") ?: @0,
            @"is_me":@(me),@"_object_id":oid} mutableCopy];
    }
    // Some versions omit the owner from sharees. This is a stored owner ID,
    // not an invented participant or an invitation to a new person.
    if (ownerString.length && !records[ownerString]) {
        records[ownerString] = [@{@"id":ownerString,@"name":text(optional(list,@"sharedOwnerName")),
            @"address":text(optional(list,@"sharedOwnerAddress")),@"access_level":@0,
            @"is_me":@(owned),@"_object_id":ownerID} mutableCopy];
    }
    NSArray *keys = [[records allKeys] sortedArrayUsingSelector:@selector(compare:)];
    NSMutableArray *result = [NSMutableArray array];
    for (NSString *key in keys) [result addObject:records[key]];
    return result;
}
static NSDictionary *publicParticipant(NSDictionary *record) {
    NSMutableDictionary *out = [record mutableCopy]; [out removeObjectForKey:@"_object_id"]; return out;
}
static NSArray *publicRoster(NSArray *roster) {
    NSMutableArray *out = [NSMutableArray array];
    for (NSDictionary *record in roster) [out addObject:publicParticipant(record)];
    return out;
}
static id currentAssigneeID(id reminder) {
    id assignment = get(get(reminder,@"assignmentContext"),@"currentAssignment");
    return assignment ? get(assignment,@"assigneeID") : nil;
}
static NSArray *sections(id list) {
    id value = withError(get(list,@"store"),@"fetchListSectionsWithListObjectID:error:",get(list,@"objectID"));
    if (![value isKindOfClass:NSArray.class]) fail(@"Unexpected section response");
    return value;
}
static NSDictionary *publicSection(id section) {
    return @{@"id":objectIDString(get(section,@"objectID")),@"name":text(get(section,@"displayName"))};
}
static NSMutableDictionary *metadata(id reminder, id list) {
    NSMutableDictionary *out = [@{@"assigned_to":NSNull.null,@"assignment_available":@NO} mutableCopy];
    @try {
        id assignee = currentAssigneeID(reminder);
        if (assignee) {
            NSString *identifier = objectIDString(assignee);
            if (!identifier.length) fail(@"Assignment has no usable ID");
            NSDictionary *person = nil;
            for (NSDictionary *p in participants(list)) if (equalText(p[@"id"],identifier)) { person = publicParticipant(p); break; }
            // Preserve orphaned assignment IDs rather than pretending unassigned.
            out[@"assigned_to"] = person ?: @{@"id":identifier,@"name":@"",@"is_me":@NO,@"access_level":@0};
        }
        out[@"assignment_available"] = @YES;
    } @catch (NSException *e) { out[@"assignment_error"] = e.reason ?: @"Assignment metadata unavailable"; }
    return out;
}
static void requireWritable(EKCalendar *calendar, id list) {
    if (!calendar.allowsContentModifications || [get(list,@"daIsReadOnly") boolValue] || [get(list,@"daIsImmutable") boolValue])
        fail(@"This list is read-only");
}

static id handle(NSDictionary *request) {
    NSString *op = text(request[@"op"]);
    dlopen("/System/Library/PrivateFrameworks/ReminderKit.framework/ReminderKit",RTLD_NOW|RTLD_LOCAL);
    if ([op isEqualToString:@"diagnostics"]) {
        return @{@"macos":NSProcessInfo.processInfo.operatingSystemVersionString,
            @"reminders_authorization_status":@([EKEventStore authorizationStatusForEntityType:EKEntityTypeReminder]),
            @"assignment_read_api":@([NSClassFromString(@"REMReminderAssignmentContext") instancesRespondToSelector:NSSelectorFromString(@"currentAssignment")]),
            @"assignment_write_api":@([NSClassFromString(@"REMReminderAssignmentContextChangeItem") instancesRespondToSelector:NSSelectorFromString(@"addAssignmentWithAssigneeID:originatorID:status:")]),
            @"section_list_api":@([NSClassFromString(@"REMStore") instancesRespondToSelector:NSSelectorFromString(@"fetchListSectionsWithListObjectID:error:")]),
            @"section_membership_supported":@NO,
            @"permission_requested":@NO};
    }
    if ([EKEventStore authorizationStatusForEntityType:EKEntityTypeReminder] != EKAuthorizationStatusAuthorized)
        fail(@"Reminders access is not authorized. Run rem lists from Terminal and approve access; see rem doctor and docs/troubleshooting.md");
    if ([op isEqualToString:@"section-move"] || [op isEqualToString:@"section-delete"]) fail(@"Section movement and deletion are not implemented");
    EKEventStore *store = [EKEventStore new];
    if ([op isEqualToString:@"participants"] || [op isEqualToString:@"sections"] || [op hasPrefix:@"section-"]) {
        EKCalendar *calendar = findList(store,text(request[@"list"]));
        id list = backingObject(calendar,@"REMList");
        if ([op isEqualToString:@"participants"]) return publicRoster(participants(list));
        NSArray *existing = sections(list);
        if ([op isEqualToString:@"sections"]) {
            NSMutableArray *out = [NSMutableArray array];
            for (id section in existing) [out addObject:publicSection(section)];
            return out;
        }
        requireWritable(calendar,list);
        NSString *name = text(request[@"name"]);
        NSString *newName = [op isEqualToString:@"section-create"] ? name : text(request[@"new_name"]);
        id match = nil;
        for (id s in existing) if (equalText(objectIDString(get(s,@"objectID")),name)) { match = s; break; }
        if (!match) for (id s in existing) if (equalText(text(get(s,@"displayName")),name)) {
            if (match) fail(@"Ambiguous section name; use the exact section ID"); match = s;
        }
        if ([op isEqualToString:@"section-delete"] || [op isEqualToString:@"section-move"])
            fail(@"Section deletion/movement is not available until native membership preservation is implemented; nothing changed");
        if (![op isEqualToString:@"section-create"] && ![op isEqualToString:@"section-rename"]) fail(@"Unsupported section operation");
        for (id s in existing) if (s != match && equalText(text(get(s,@"displayName")),newName)) fail(@"A section with that name already exists");
        if ([op isEqualToString:@"section-create"] && match) fail(@"A section with that name already exists");
        if ([op isEqualToString:@"section-rename"] && !match) fail(@"Section not found");
        if (!newName.length) fail(@"Section name is required");
        id nativeStore = get(list,@"store");
        id saveRequest = newObject(@"REMSaveRequest",@"initWithStore:",@[nativeStore]);
        id changed;
        if ([op isEqualToString:@"section-create"]) {
            id listChange = invoke(saveRequest,@"updateList:",@[list]);
            id ctx = get(listChange,@"sectionsContextChangeItem");
            changed = invoke(saveRequest,@"addListSectionWithDisplayName:toListSectionContextChangeItem:",@[newName,ctx]);
        } else {
            changed = invoke(saveRequest,@"updateListSection:",@[match]);
            [changed setValue:newName forKey:@"displayName"];
        }
        if (!changed) fail(@"Could not prepare section change; nothing saved");
        id sectionID = get(changed,@"objectID");
        save(saveRequest);
        @try {
            id fresh = withError(nativeStore,@"fetchListSectionWithObjectID:error:",sectionID);
            if (![text(get(fresh,@"displayName")) isEqualToString:newName]) fail(@"Section name did not match");
            return publicSection(fresh);
        } @catch (NSException *e) { fail([@"Section save succeeded but verification failed; inspect Reminders before retrying: " stringByAppendingString:e.reason]); }
    }
    NSArray<EKReminder *> *all = fetchReminders(store,nil);
    if ([op isEqualToString:@"metadata"]) {
        NSMutableDictionary *out = [NSMutableDictionary dictionary];
        NSMutableDictionary *byID = [NSMutableDictionary dictionary];
        for (EKReminder *r in all) byID[r.calendarItemIdentifier.lowercaseString] = r;
        for (NSString *identifier in request[@"ids"]) {
            @try {
                EKReminder *ek = byID[normalizeID(identifier).lowercaseString];
                if (!ek) fail(@"Reminder no longer exists");
                out[identifier] = metadata(backingObject(ek,@"REMReminder"),backingObject(ek.calendar,@"REMList"));
            } @catch (NSException *e) {
                out[identifier] = @{@"assigned_to":NSNull.null,@"assignment_available":@NO,
                    @"assignment_error":e.reason ?: @"Unavailable"};
            }
        }
        return out;
    }
    EKReminder *ek = findReminder(all,text(request[@"id"]));
    id reminder = backingObject(ek,@"REMReminder");
    id list = backingObject(ek.calendar,@"REMList");
    NSArray *roster = participants(list);
    if ([op isEqualToString:@"roster"]) {
        if (![get(list,@"isShared") boolValue]) fail(@"Assignments require a shared list");
        return @{@"people":publicRoster(roster),@"list_id":ek.calendar.calendarIdentifier,@"reminder_id":ek.calendarItemIdentifier};
    }
    if (![op isEqualToString:@"assign"]) fail(@"Unsupported operation");
    if (!equalText(ek.calendarItemIdentifier, text(request[@"id"]))) fail(@"Resolved reminder no longer exists; nothing changed");
    if (![get(list,@"isShared") boolValue]) fail(@"Assignments require a shared list");
    if (![ek.calendar.calendarIdentifier isEqualToString:request[@"list_id"]]) fail(@"Reminder moved to another list; re-run after reviewing its participants");
    requireWritable(ek.calendar,list);
    BOOL clear = [request[@"clear"] boolValue];
    NSString *targetID = text(request[@"participant_id"]);
    NSDictionary *target = nil, *me = nil;
    for (NSDictionary *p in roster) {
        if (equalText(p[@"id"],targetID)) target = p;
        if ([p[@"is_me"] boolValue]) { if (me) fail(@"Native current-user identity is ambiguous"); me = p; }
    }
    if (!clear && !target) fail(@"Participant is no longer in this shared list");
    NSString *oldID = objectIDString(currentAssigneeID(reminder));
    if ((clear && !oldID.length) || (!clear && equalText(oldID,targetID))) return metadata(reminder,list);
    // Do not fabricate an originator or substitute the assignee for the caller.
    if (!clear && !me) fail(@"Cannot resolve the current user's native participant ID on this account; nothing changed");
    if (!clear && (![target[@"_object_id"] isKindOfClass:NSClassFromString(@"REMObjectID")] ||
        ![me[@"_object_id"] isKindOfClass:NSClassFromString(@"REMObjectID")]))
        fail(@"Native participant identity type changed; nothing saved");
    id nativeStore = get(reminder,@"store");
    id saveRequest = newObject(@"REMSaveRequest",@"initWithStore:",@[nativeStore]);
    id change = invoke(saveRequest,@"updateReminder:",@[reminder]);
    id context = get(change,@"assignmentContext");
    invoke(context,@"removeAllAssignments",@[]);
    if (!clear) {
        // Status 0 is provisional: synthetic native storage accepts it, but
        // that does not establish app notification/sync semantics. CLI writes
        // require --experimental until a real shared-account test verifies it.
        invoke(context,@"addAssignmentWithAssigneeID:originatorID:status:",@[target[@"_object_id"],me[@"_object_id"],@0]);
        if (!equalText(objectIDString(get(get(context,@"currentAssignment"),@"assigneeID")),targetID))
            fail(@"Native assignment state was not accepted; nothing saved");
    }
    save(saveRequest);
    @try {
        id fresh = withError(nativeStore,@"fetchReminderWithObjectID:error:",get(reminder,@"objectID"));
        NSString *saved = objectIDString(currentAssigneeID(fresh));
        if (clear ? saved.length != 0 : !equalText(saved,targetID)) fail(@"Assignment did not match");
        return metadata(fresh,list);
    } @catch (NSException *e) { fail([@"Assignment save succeeded but verification failed; inspect Reminders before retrying: " stringByAppendingString:e.reason]); }
    return nil;
}

char *rem_collaboration_call(const char *requestJSON) {
    static dispatch_queue_t queue;
    static dispatch_once_t once;
    dispatch_once(&once,^{ queue = dispatch_queue_create("rem.collaboration",DISPATCH_QUEUE_SERIAL); });
    __block char *result = NULL;
    dispatch_sync(queue,^{
        @autoreleasepool {
            NSDictionary *envelope;
            @try {
                NSData *data = [NSData dataWithBytes:requestJSON length:strlen(requestJSON)];
                id request = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
                if (![request isKindOfClass:NSDictionary.class]) fail(@"Invalid bridge request");
                envelope = @{@"result":handle(request) ?: NSNull.null};
            } @catch (NSException *e) { envelope = @{@"error":e.reason ?: @"Native collaboration operation failed"}; }
            NSData *data = [NSJSONSerialization dataWithJSONObject:envelope options:0 error:nil];
            result = strdup(data ? [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding].UTF8String : "{\"error\":\"Could not encode native response\"}");
        }
    });
    return result;
}
