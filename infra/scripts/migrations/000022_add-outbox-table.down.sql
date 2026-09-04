alter table processed_events
drop constraint processed_events_outbox_id_fk;

drop index processed_events_service_idx;

alter table processed_events
drop constraint processed_events_unique_delivery;

drop table processed_events;

drop index outbox_tracing_id_idx;

drop index outbox_polling_idx;

drop table outbox;

drop type outbox_status;