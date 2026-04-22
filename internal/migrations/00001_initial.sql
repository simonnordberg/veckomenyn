-- +goose Up

-- family composition ---------------------------------------------------------

CREATE TABLE family_members (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    display_name TEXT NOT NULL,
    birth_date DATE,
    role TEXT NOT NULL CHECK (role IN ('adult', 'child')),
    schedule_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    notes_md TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE dietary_constraints (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    family_member_id BIGINT REFERENCES family_members(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('allergy', 'dislike', 'preference')),
    label TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_dietary_constraints_member ON dietary_constraints(family_member_id);

-- shopping and cooking rules -------------------------------------------------

CREATE TABLE pantry_items (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE brand_preferences (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category TEXT NOT NULL,
    brand TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (category, brand)
);

CREATE TABLE sourcing_rules (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ingredient_category TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('willys', 'butcher', 'fishmonger', 'bakery', 'other')),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE shopping_rules (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    body_md TEXT NOT NULL,
    priority INT NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cooking_principles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category TEXT NOT NULL,
    body_md TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- recipes --------------------------------------------------------------------

CREATE TABLE dishes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    cuisine TEXT,
    recipe_md TEXT NOT NULL DEFAULT '',
    servings INT NOT NULL DEFAULT 4,
    tags_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    last_made_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_dishes_last_made ON dishes(last_made_at DESC NULLS LAST);
CREATE INDEX idx_dishes_tags ON dishes USING gin(tags_json);

CREATE TABLE ingredients (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    unit TEXT,
    default_source TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE dish_ingredients (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dish_id BIGINT NOT NULL REFERENCES dishes(id) ON DELETE CASCADE,
    ingredient_id BIGINT REFERENCES ingredients(id) ON DELETE SET NULL,
    qty NUMERIC(10, 3),
    unit TEXT,
    prep TEXT,
    optional BOOLEAN NOT NULL DEFAULT false,
    sort_order INT NOT NULL DEFAULT 0
);
CREATE INDEX idx_dish_ingredients_dish ON dish_ingredients(dish_id);

-- weekly planning ------------------------------------------------------------

CREATE TABLE weeks (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    iso_week TEXT NOT NULL UNIQUE, -- e.g. "2026-W17"
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'cart_built', 'ordered', 'archived')),
    notes_md TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_weeks_start_date ON weeks(start_date DESC);

CREATE TABLE week_exceptions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    week_id BIGINT NOT NULL REFERENCES weeks(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_week_exceptions_week ON week_exceptions(week_id);

CREATE TABLE week_dinners (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    week_id BIGINT NOT NULL REFERENCES weeks(id) ON DELETE CASCADE,
    day_date DATE NOT NULL,
    dish_id BIGINT REFERENCES dishes(id) ON DELETE SET NULL,
    servings INT NOT NULL DEFAULT 4,
    sourcing_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    notes TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_week_dinners_week ON week_dinners(week_id, day_date);

-- shopping cart --------------------------------------------------------------

CREATE TABLE willys_products (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    last_price NUMERIC(10, 2),
    category_path TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cart_items (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    week_id BIGINT REFERENCES weeks(id) ON DELETE CASCADE,
    product_code TEXT NOT NULL,
    qty NUMERIC(10, 3) NOT NULL DEFAULT 1,
    reason_md TEXT NOT NULL DEFAULT '',
    added_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    committed BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX idx_cart_items_week ON cart_items(week_id);

-- feedback -------------------------------------------------------------------

CREATE TABLE retrospectives (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    week_id BIGINT NOT NULL REFERENCES weeks(id) ON DELETE CASCADE,
    notes_md TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_retrospectives_week ON retrospectives(week_id);

CREATE TABLE dish_ratings (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    week_dinner_id BIGINT NOT NULL REFERENCES week_dinners(id) ON DELETE CASCADE,
    rating TEXT CHECK (rating IN ('loved', 'liked', 'meh', 'disliked')),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_dish_ratings_dinner ON dish_ratings(week_dinner_id);

-- chat -----------------------------------------------------------------------

CREATE TABLE conversations (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    week_id BIGINT REFERENCES weeks(id) ON DELETE SET NULL,
    title TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_conversations_updated ON conversations(updated_at DESC);

CREATE TABLE messages (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'tool')),
    content_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at);

-- willys session (shared family account) -------------------------------------

CREATE TABLE willys_session (
    id INT PRIMARY KEY CHECK (id = 1),
    cookies_bytea BYTEA,
    csrf TEXT,
    username TEXT,
    refreshed_at TIMESTAMPTZ
);

-- updated_at triggers --------------------------------------------------------

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER tg_family_members_updated BEFORE UPDATE ON family_members FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER tg_shopping_rules_updated BEFORE UPDATE ON shopping_rules FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER tg_cooking_principles_updated BEFORE UPDATE ON cooking_principles FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER tg_dishes_updated BEFORE UPDATE ON dishes FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER tg_weeks_updated BEFORE UPDATE ON weeks FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER tg_conversations_updated BEFORE UPDATE ON conversations FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- +goose Down

DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS conversations CASCADE;
DROP TABLE IF EXISTS dish_ratings CASCADE;
DROP TABLE IF EXISTS retrospectives CASCADE;
DROP TABLE IF EXISTS cart_items CASCADE;
DROP TABLE IF EXISTS willys_products CASCADE;
DROP TABLE IF EXISTS week_dinners CASCADE;
DROP TABLE IF EXISTS week_exceptions CASCADE;
DROP TABLE IF EXISTS weeks CASCADE;
DROP TABLE IF EXISTS dish_ingredients CASCADE;
DROP TABLE IF EXISTS ingredients CASCADE;
DROP TABLE IF EXISTS dishes CASCADE;
DROP TABLE IF EXISTS cooking_principles CASCADE;
DROP TABLE IF EXISTS shopping_rules CASCADE;
DROP TABLE IF EXISTS sourcing_rules CASCADE;
DROP TABLE IF EXISTS brand_preferences CASCADE;
DROP TABLE IF EXISTS pantry_items CASCADE;
DROP TABLE IF EXISTS dietary_constraints CASCADE;
DROP TABLE IF EXISTS family_members CASCADE;
DROP TABLE IF EXISTS willys_session CASCADE;
DROP FUNCTION IF EXISTS touch_updated_at();
