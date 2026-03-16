ALTER TABLE resume_builders
  ADD COLUMN layout_mode VARCHAR(20) NOT NULL DEFAULT 'single'
    CHECK (layout_mode IN ('single', 'double-left', 'double-right', 'custom')),
  ADD COLUMN sidebar_width SMALLINT NOT NULL DEFAULT 35
    CHECK (sidebar_width BETWEEN 25 AND 50);

ALTER TABLE resume_section_orders
  ADD COLUMN column_placement VARCHAR(10) NOT NULL DEFAULT 'main'
    CHECK (column_placement IN ('main', 'sidebar'));
