ALTER TABLE notifications
ADD COLUMN event_id uuid;

UPDATE notifications
SET event_id = gen_random_uuid()
WHERE event_id IS NULL;

ALTER TABLE notifications
ALTER COLUMN event_id SET NOT NULL;

CREATE UNIQUE INDEX idx_notifications_event_id
ON notifications(event_id);