# Design Document: Task Due Dates & Notification System

## 1. Database Schema Changes

### 1.1 Task Table Modification
Add `due_at` column to the existing `tasks` table.

```sql
ALTER TABLE tasks ADD COLUMN due_at DATETIME;
```

### 1.2 Notification Settings Table
Store per-user configuration for notifications.

```sql
CREATE TABLE notification_settings (
    user_id INTEGER PRIMARY KEY,
    reminder_delta_minutes INTEGER DEFAULT 60, -- Notify X mins before deadline
    snooze_default_minutes INTEGER DEFAULT 15, -- Default snooze duration
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);
```

### 1.3 Notification State Tracking
Track which notifications have been sent to avoid duplicates and handle snoozing.

```sql
CREATE TABLE task_notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL, -- Recipient
    notification_type TEXT NOT NULL CHECK(notification_type IN ('before_deadline', 'on_deadline', 'snooze')),
    scheduled_at DATETIME NOT NULL, -- When it should be sent
    sent_at DATETIME, -- When it was actually sent (NULL if pending)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE INDEX idx_task_notifications_scheduled_pending ON task_notifications(scheduled_at) WHERE sent_at IS NULL;
```

### 1.4 Migration Strategy
1.  Create new tables (`notification_settings`, `task_notifications`).
2.  Add `due_at` column to `tasks`.
3.  Insert default `notification_settings` for existing users (optional, or handle lazily in code).

## 2. Notification Scheduling Logic

### 2.1 Trigger Generation
When a task is created or updated with a `due_at`:
1.  Calculate notification times for all group members (or just the assignee if we had one, but currently tasks are group-wide). *Assumption: Notify all group members who have notifications enabled.*
2.  Insert rows into `task_notifications`:
    -   **Before Deadline**: `due_at - user.reminder_delta_minutes`
    -   **On Deadline**: `due_at`

### 2.2 Polling Worker
A background goroutine (Ticker) running every minute:
1.  Query `task_notifications`:
    ```sql
    SELECT * FROM task_notifications 
    WHERE sent_at IS NULL 
    AND scheduled_at <= CURRENT_TIMESTAMP
    ```
2.  For each record:
    -   Fetch Task and User details.
    -   Send Telegram message via Bot.
    -   Update `sent_at = CURRENT_TIMESTAMP`.

### 2.3 Snooze Logic
When a user clicks "Snooze":
1.  Calculate new time: `NOW() + snooze_minutes`.
2.  Insert new row into `task_notifications`:
    -   `task_id`: current task
    -   `user_id`: user who snoozed
    -   `notification_type`: 'snooze'
    -   `scheduled_at`: calculated time

## 3. Bot Integration Points

### 3.1 Message Format
```
⏰ Reminder: [Task Title]
Due: [Due Date/Time]
[Description if exists]

Buttons:
[ ✅ Done ]
[ 💤 Snooze (15m) ]
[ ⏰ Remind Later... ]
```

### 3.2 Callback Handlers

#### `task:done:{taskID}`
-   Same logic as existing `/tasks` completion.
-   Mark task completed.
-   (Optional) Delete pending notifications for this task.

#### `task:snooze:{taskID}:{minutes}`
-   If `minutes` is provided (e.g., "15"), schedule new 'snooze' notification.
-   Ack callback with "Snoozed for 15m".
-   Delete/Edit original message to remove buttons or update status.

#### `task:remind_later:{taskID}`
-   Edit message to show duration options:
    -   [ 30 mins ]
    -   [ 1 hour ]
    -   [ 3 hours ]
    -   [ Tomorrow morning ]
    -   [ ⬅️ Back ]
-   Clicking an option triggers `task:snooze:{taskID}:{calculated_minutes}`.

## 4. Configuration Model

### 4.1 Interfaces

**Core Models (`internal/core/models.go`)**:
```go
type NotificationSettings struct {
    UserID               int64
    ReminderDeltaMinutes int
    SnoozeDefaultMinutes int
}

type TaskNotification struct {
    ID               int64
    TaskID           int64
    UserID           int64
    NotificationType string // 'before_deadline', 'on_deadline', 'snooze'
    ScheduledAt      time.Time
    SentAt           *time.Time
}
```

**Store Interface (`internal/core/service.go`)**:
```go
type Store interface {
    // ... existing methods ...
    
    // Notification Settings
    GetNotificationSettings(userID int64) (*NotificationSettings, error)
    UpdateNotificationSettings(settings *NotificationSettings) error
    
    // Notification Queue
    CreateTaskNotification(n *TaskNotification) error
    GetPendingNotifications() ([]*TaskNotification, error)
    MarkNotificationSent(id int64) error
    DeletePendingNotificationsByTaskID(taskID int64) error
}
```

### 4.2 Defaults
-   `ReminderDeltaMinutes`: 60 (1 hour)
-   `SnoozeDefaultMinutes`: 15

## 5. Implementation Steps (Summary)

1.  **Store**: Implement schema migrations and new store methods in `sqlite.go` / new `notifications.go`.
2.  **Service**: Add `ScheduleTaskNotifications(task *Task)` method. Call this on Task Create/Update.
3.  **Worker**: Implement `NotificationWorker` in `internal/core` or `internal/bot` that polls and sends.
4.  **Bot**: Implement new callback handlers and message templates.