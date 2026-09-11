-- Rename existing workshop lines + add new uchastka names (catalog only)
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/added/121_lines_uchastka_names.sql

BEGIN;

-- Existing operational lines (IDs unchanged)
UPDATE lines.lines_list
SET name = 'Boshlang''ich yig''uv uchastkasi'
WHERE line_id = 1;

UPDATE lines.lines_list
SET name = 'Eshiklarni kesish va formalash uchastkasi'
WHERE line_id = 11;

UPDATE lines.lines_list
SET name = 'Yakuniy yig''uv uchastkasi'
WHERE line_id = 12;

-- New uchastkalar (name only; no routes/features yet)
INSERT INTO lines.lines_list (id, line_id, name, status, folder_name)
OVERRIDING SYSTEM VALUE
SELECT 13, 13, 'Vakuumda formalash uchastkasi', true, 'vakuum'
WHERE NOT EXISTS (SELECT 1 FROM lines.lines_list WHERE line_id = 13);

INSERT INTO lines.lines_list (id, line_id, name, status, folder_name)
OVERRIDING SYSTEM VALUE
SELECT 14, 14, 'U-shell uchastkasi', true, 'u_shell'
WHERE NOT EXISTS (SELECT 1 FROM lines.lines_list WHERE line_id = 14);

INSERT INTO lines.lines_list (id, line_id, name, status, folder_name)
OVERRIDING SYSTEM VALUE
SELECT 15, 15, 'Shelkografiya uchastkasi', true, 'shelkografiya'
WHERE NOT EXISTS (SELECT 1 FROM lines.lines_list WHERE line_id = 15);

INSERT INTO lines.lines_list (id, line_id, name, status, folder_name)
OVERRIDING SYSTEM VALUE
SELECT 16, 16, 'Shkafga PPU quyish uchastkasi', true, 'ppu'
WHERE NOT EXISTS (SELECT 1 FROM lines.lines_list WHERE line_id = 16);

INSERT INTO lines.lines_list (id, line_id, name, status, folder_name)
OVERRIDING SYSTEM VALUE
SELECT 17, 17, 'Agregat yig''uv uchastkasi', true, 'agregat'
WHERE NOT EXISTS (SELECT 1 FROM lines.lines_list WHERE line_id = 17);

INSERT INTO lines.lines_list (id, line_id, name, status, folder_name)
OVERRIDING SYSTEM VALUE
SELECT 18, 18, 'Kondensator va bug''lantiruvchi yig''ish uchastkasi', true, 'kondensator'
WHERE NOT EXISTS (SELECT 1 FROM lines.lines_list WHERE line_id = 18);

INSERT INTO lines.lines_list (id, line_id, name, status, folder_name)
OVERRIDING SYSTEM VALUE
SELECT 19, 19, 'Laboratoriya tekshiruv uchastkasi', true, 'lab_tekshiruv'
WHERE NOT EXISTS (SELECT 1 FROM lines.lines_list WHERE line_id = 19);

COMMIT;
