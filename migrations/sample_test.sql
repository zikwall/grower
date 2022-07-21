CREATE TABLE access_log (
    remote_addr String,
    remote_user String,
    time_local DateTime,
    request String,
    request_method String,
    status Int32,
    bytes_sent Int32,
    http_referer String,
    http_user_agent String,
    request_time Float32,
    https FixedString(2),
    insert_date Date DEFAULT toDate(time_local),
    custom_field Int32,
    custom_time_field Datetime
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(insert_date)
ORDER BY (status, insert_date);