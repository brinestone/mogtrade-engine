-- DOWN MIGRATION: Revert the refactored executions table back to the original schema

DROP TABLE IF EXISTS "executions" CASCADE;

DROP TYPE IF EXISTS "execution_status";

CREATE TYPE "execution_status" AS ENUM (
	'failed',
	'filled'
);

CREATE TABLE "executions" (
	"id" varchar(26) PRIMARY KEY,
	"order" varchar(26) NOT NULL,
	"price" numeric(18, 4) NOT NULL,
	"quantity" numeric(18, 4) NOT NULL,
	"order_side" order_side NOT NULL,
	"order_status" order_status NOT NULL,
	"order_type" order_type NOT NULL,
	"symbol" text NOT NULL,
	"user" varchar(26),
	"fee_currency" text NOT NULL,
	"fee_rate" numeric(18, 4) NOT NULL,
	"fee_amount" numeric(18, 4) NOT NULL,
	"exec_status" execution_status NOT NULL,
	"recorded_at" timestamp with time zone DEFAULT now() NOT NULL,
	"tracing_id" varchar(26) NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL,
	"idempotency_token" text NOT NULL
);

CREATE INDEX "executions_fee_currency_idx" ON "executions" ("fee_currency");
CREATE UNIQUE INDEX "executions_idempotency_token_order_uidx" ON "executions" ("idempotency_token","order");
CREATE INDEX "executions_order_idx" ON "executions" ("order");
CREATE INDEX "executions_symbol_idx" ON "executions" ("symbol");
CREATE UNIQUE INDEX "executions_tracing_id_idx" ON "executions" ("tracing_id");
CREATE INDEX "executions_user_idx" ON "executions" ("user");

ALTER TABLE "executions" ADD CONSTRAINT "executions_order_orders_id_fk" FOREIGN KEY ("order") REFERENCES "orders"("id") ON DELETE CASCADE;
ALTER TABLE "executions" ADD CONSTRAINT "executions_user_user_id_fk" FOREIGN KEY ("user") REFERENCES "user"("id") ON DELETE SET NULL;