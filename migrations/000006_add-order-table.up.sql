CREATE TABLE
    orders (
        user_id varchar(26),
        symbol text NOT NULL,
        side order_side NOT NULL,
        order_type order_type NOT NULL,
        quantity FLOAT NOT NULL,
        limit_price FLOAT,
        stop_price FLOAT,
        id varchar(26) PRIMARY KEY NOT NULL,
        client_order_id TEXT,
        status order_status DEFAULT 'pending',
        filled_quantity FLOAT DEFAULT 0.0,
        average_fill_price FLOAT,
        created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
    );

CREATE INDEX "orders_user_id_idx" ON "orders" ("user_id");

CREATE INDEX "orders_client_order_id_idx" ON "orders" ("client_order_id")
WHERE
    ("client_order_id" IS NOT NULL);

CREATE UNIQUE INDEX "orders_user_id_client_order_id_uidx" ON "orders" ("user_id", "client_order_id")
WHERE
    (
        "user_id" IS NOT NULL
        AND "client_order_id" IS NOT NULL
    );

ALTER TABLE "orders"
ADD CONSTRAINT "orders_user_id_user_id_fk" FOREIGN KEY ("user_id") REFERENCES "user" ("id") ON DELETE SET NULL;