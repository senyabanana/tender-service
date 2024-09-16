CREATE TABLE IF NOT EXISTS employee (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TYPE organization_type AS ENUM (
    'IE',
    'LLC',
    'JSC'
    );

CREATE TABLE IF NOT EXISTS organization (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type organization_type,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS organization_responsible (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organization(id) ON DELETE CASCADE,
    user_id UUID REFERENCES employee(id) ON DELETE CASCADE
);


CREATE TABLE IF NOT EXISTS tender (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    service_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    organization_id UUID REFERENCES organization(id) ON DELETE CASCADE,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    creator_username VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS tender_history (
    id UUID REFERENCES tender(id) ON DELETE CASCADE,
    name VARCHAR(100),
    description TEXT,
    service_type VARCHAR(50),
    status VARCHAR(50),
    organization_id UUID,
    version INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    creator_username VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS bid (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL,
    tender_id UUID REFERENCES tender(id) ON DELETE CASCADE,
    author_type VARCHAR(50) NOT NULL,
    author_id UUID NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bid_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bid_id UUID REFERENCES bid(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL,
    author_type VARCHAR(50) NOT NULL,
    author_id UUID NOT NULL,
    version INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bid_review (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bid_id UUID REFERENCES bid(id) ON DELETE CASCADE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

