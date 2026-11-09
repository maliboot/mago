create table archive_message
(
    id         integer primary key autoincrement,
    task_id    text    not null,
    task_group text    not null,
    name       text    not null,
    body       blob    not null,
    result     integer not null default 0,
    result_txt text    not null,
    ext        text    not null,
    created    text    not null default (strftime('%Y-%m-%dT%H:%M:%fZ')),
    updated    text    not null default (strftime('%Y-%m-%dT%H:%M:%fZ'))
) strict;

create trigger archive_message_updated_timestamp
    after update
    on archive_message
begin
    update archive_message set updated = strftime('%Y-%m-%dT%H:%M:%fZ') where id = old.id;
end;

CREATE INDEX IF NOT EXISTS archive_message_result_task_group_idx on archive_message (task_group, result);
CREATE INDEX IF NOT EXISTS archive_message_task_id ON archive_message (task_id);
CREATE INDEX IF NOT EXISTS idx_archive_message_result_created ON archive_message(result, created);