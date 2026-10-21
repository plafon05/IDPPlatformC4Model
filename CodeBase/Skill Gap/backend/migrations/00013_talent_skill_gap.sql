-- +goose Up
CREATE TABLE position_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE profile_requirements (
    profile_id UUID NOT NULL REFERENCES position_profiles(id) ON DELETE CASCADE,
    version INTEGER NOT NULL CHECK (version >= 1),
    competency_id UUID NOT NULL REFERENCES competencies(id),
    required_level SMALLINT NOT NULL CHECK (required_level BETWEEN 1 AND 4),
    PRIMARY KEY (profile_id, version, competency_id)
);
CREATE TABLE employee_competency_levels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES users(id),
    competency_id UUID NOT NULL REFERENCES competencies(id),
    level SMALLINT NOT NULL CHECK (level BETWEEN 1 AND 4),
    status VARCHAR(30) NOT NULL CHECK (status IN ('final','pending_confirmation','rejected')),
    method VARCHAR(30) NOT NULL CHECK (method IN ('manager_decision','idp_rule')),
    basis TEXT NOT NULL,
    set_by UUID REFERENCES users(id),
    set_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ecl_employee ON employee_competency_levels(employee_id, competency_id, set_at);

-- +goose Down
DROP INDEX IF EXISTS idx_ecl_employee;
DROP TABLE IF EXISTS employee_competency_levels;
DROP TABLE IF EXISTS profile_requirements;
DROP TABLE IF EXISTS position_profiles;
