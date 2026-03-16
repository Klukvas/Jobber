-- Resume Builder tables

-- Templates (seed data)
CREATE TABLE resume_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    is_premium BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO resume_templates (id, name, display_name, description, is_premium) VALUES
    ('00000000-0000-0000-0000-000000000001', 'professional', 'Professional', 'Traditional single-column layout with serif font', false),
    ('00000000-0000-0000-0000-000000000002', 'modern', 'Modern', 'Two-column layout with sidebar for contact and skills', false),
    ('00000000-0000-0000-0000-000000000003', 'minimal', 'Minimal', 'Clean single-column layout with sans-serif font', false);

-- Master resume builder table
CREATE TABLE resume_builders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL DEFAULT 'Untitled Resume',
    template_id UUID NOT NULL REFERENCES resume_templates(id) DEFAULT '00000000-0000-0000-0000-000000000001',
    font_family VARCHAR(100) NOT NULL DEFAULT 'Georgia',
    primary_color VARCHAR(7) NOT NULL DEFAULT '#2563eb',
    spacing SMALLINT NOT NULL DEFAULT 100 CHECK (spacing BETWEEN 50 AND 150),
    margin_top SMALLINT NOT NULL DEFAULT 40,
    margin_bottom SMALLINT NOT NULL DEFAULT 40,
    margin_left SMALLINT NOT NULL DEFAULT 40,
    margin_right SMALLINT NOT NULL DEFAULT 40,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resume_builders_user_id ON resume_builders(user_id);
CREATE TRIGGER update_resume_builders_updated_at BEFORE UPDATE ON resume_builders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Contact info (1:1)
CREATE TABLE resume_contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL UNIQUE REFERENCES resume_builders(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL DEFAULT '',
    email VARCHAR(255) NOT NULL DEFAULT '',
    phone VARCHAR(50) NOT NULL DEFAULT '',
    location VARCHAR(255) NOT NULL DEFAULT '',
    website VARCHAR(500) NOT NULL DEFAULT '',
    linkedin VARCHAR(500) NOT NULL DEFAULT '',
    github VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_resume_contacts_updated_at BEFORE UPDATE ON resume_contacts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Summary (1:1)
CREATE TABLE resume_summaries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL UNIQUE REFERENCES resume_builders(id) ON DELETE CASCADE,
    content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_resume_summaries_updated_at BEFORE UPDATE ON resume_summaries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Experiences (1:N)
CREATE TABLE resume_experiences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL REFERENCES resume_builders(id) ON DELETE CASCADE,
    company VARCHAR(255) NOT NULL DEFAULT '',
    position VARCHAR(255) NOT NULL DEFAULT '',
    location VARCHAR(255) NOT NULL DEFAULT '',
    start_date VARCHAR(20) NOT NULL DEFAULT '',
    end_date VARCHAR(20) NOT NULL DEFAULT '',
    is_current BOOLEAN NOT NULL DEFAULT false,
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resume_experiences_builder ON resume_experiences(resume_builder_id, sort_order);
CREATE TRIGGER update_resume_experiences_updated_at BEFORE UPDATE ON resume_experiences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Educations (1:N)
CREATE TABLE resume_educations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL REFERENCES resume_builders(id) ON DELETE CASCADE,
    institution VARCHAR(255) NOT NULL DEFAULT '',
    degree VARCHAR(255) NOT NULL DEFAULT '',
    field_of_study VARCHAR(255) NOT NULL DEFAULT '',
    start_date VARCHAR(20) NOT NULL DEFAULT '',
    end_date VARCHAR(20) NOT NULL DEFAULT '',
    is_current BOOLEAN NOT NULL DEFAULT false,
    gpa VARCHAR(20) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resume_educations_builder ON resume_educations(resume_builder_id, sort_order);
CREATE TRIGGER update_resume_educations_updated_at BEFORE UPDATE ON resume_educations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Skills (1:N)
CREATE TABLE resume_skills (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL REFERENCES resume_builders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT '',
    level VARCHAR(50) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resume_skills_builder ON resume_skills(resume_builder_id, sort_order);
CREATE TRIGGER update_resume_skills_updated_at BEFORE UPDATE ON resume_skills
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Languages (1:N)
CREATE TABLE resume_languages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL REFERENCES resume_builders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT '',
    proficiency VARCHAR(50) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resume_languages_builder ON resume_languages(resume_builder_id, sort_order);
CREATE TRIGGER update_resume_languages_updated_at BEFORE UPDATE ON resume_languages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Certifications (1:N)
CREATE TABLE resume_certifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL REFERENCES resume_builders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT '',
    issuer VARCHAR(255) NOT NULL DEFAULT '',
    issue_date VARCHAR(20) NOT NULL DEFAULT '',
    expiry_date VARCHAR(20) NOT NULL DEFAULT '',
    url VARCHAR(500) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resume_certifications_builder ON resume_certifications(resume_builder_id, sort_order);
CREATE TRIGGER update_resume_certifications_updated_at BEFORE UPDATE ON resume_certifications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Projects (1:N)
CREATE TABLE resume_projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL REFERENCES resume_builders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT '',
    url VARCHAR(500) NOT NULL DEFAULT '',
    start_date VARCHAR(20) NOT NULL DEFAULT '',
    end_date VARCHAR(20) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resume_projects_builder ON resume_projects(resume_builder_id, sort_order);
CREATE TRIGGER update_resume_projects_updated_at BEFORE UPDATE ON resume_projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Volunteering (1:N)
CREATE TABLE resume_volunteering (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL REFERENCES resume_builders(id) ON DELETE CASCADE,
    organization VARCHAR(255) NOT NULL DEFAULT '',
    role VARCHAR(255) NOT NULL DEFAULT '',
    start_date VARCHAR(20) NOT NULL DEFAULT '',
    end_date VARCHAR(20) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resume_volunteering_builder ON resume_volunteering(resume_builder_id, sort_order);
CREATE TRIGGER update_resume_volunteering_updated_at BEFORE UPDATE ON resume_volunteering
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Custom sections (1:N, premium)
CREATE TABLE resume_custom_sections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL REFERENCES resume_builders(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resume_custom_sections_builder ON resume_custom_sections(resume_builder_id, sort_order);
CREATE TRIGGER update_resume_custom_sections_updated_at BEFORE UPDATE ON resume_custom_sections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Section ordering and visibility
CREATE TABLE resume_section_orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resume_builder_id UUID NOT NULL REFERENCES resume_builders(id) ON DELETE CASCADE,
    section_key VARCHAR(50) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_visible BOOLEAN NOT NULL DEFAULT true,
    UNIQUE (resume_builder_id, section_key)
);

CREATE INDEX idx_resume_section_orders_builder ON resume_section_orders(resume_builder_id, sort_order);

-- Cover letters
CREATE TABLE cover_letters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resume_builder_id UUID REFERENCES resume_builders(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL DEFAULT 'Untitled Cover Letter',
    template VARCHAR(50) NOT NULL DEFAULT 'professional',
    recipient_name VARCHAR(255) NOT NULL DEFAULT '',
    recipient_title VARCHAR(255) NOT NULL DEFAULT '',
    company_name VARCHAR(255) NOT NULL DEFAULT '',
    company_address VARCHAR(500) NOT NULL DEFAULT '',
    greeting VARCHAR(255) NOT NULL DEFAULT '',
    paragraphs TEXT[] NOT NULL DEFAULT '{}',
    closing VARCHAR(255) NOT NULL DEFAULT '',
    font_family VARCHAR(100) NOT NULL DEFAULT 'Georgia',
    primary_color VARCHAR(7) NOT NULL DEFAULT '#2563eb',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_cover_letters_user_id ON cover_letters(user_id);
CREATE TRIGGER update_cover_letters_updated_at BEFORE UPDATE ON cover_letters
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Content library
CREATE TABLE content_library (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_content_library_user_id ON content_library(user_id);
CREATE INDEX idx_content_library_category ON content_library(user_id, category);
CREATE TRIGGER update_content_library_updated_at BEFORE UPDATE ON content_library
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
