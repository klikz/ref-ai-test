-- auth.routes: faol API route commentlari (routecomments.go o'rniga)
-- Apply:
--   psql "host=localhost dbname=ac2 user=postgres password=postgres sslmode=disable" -f migrations/115_auth_routes.sql

BEGIN;

CREATE TEMP TABLE tmp_route_comments (
    route TEXT PRIMARY KEY,
    comment TEXT NOT NULL
) ON COMMIT DROP;

INSERT INTO tmp_route_comments (route, comment) VALUES
    ('/api/user/create', 'Foydalanuvchi yaratish'),
    ('/api/user/update', 'Foydalanuvchi ma''lumotlari yoki parolini yangilash'),
    ('/api/user/delete', 'Foydalanuvchini deaktiv qilish'),
    ('/api/user/getall', 'Foydalanuvchilar ro''yxatini ko''rish'),
    ('/api/user/:id', 'Foydalanuvchi profili va ruxsatlarini ko''rish'),
    ('/api/user/permission/:id', 'Foydalanuvchi route ruxsatlarini o''zgartirish'),
    ('/api/serial/info', 'Serial/kompressor bo''yicha mahsulot, params va skan surati'),
    ('/api/lab/info', 'Laboratoriya (VTM) BxData — serial bo''yicha test natijalari'),
    ('/api/tech/models/add', 'Modellarni Excel orqali qo''shish'),
    ('/api/tech/models/all', 'Modellar katalogini ko''rish'),
    ('/api/tech/models/update', 'Model ma''lumotlarini Excel orqali yangilash'),
    ('/api/tech/models/status', 'Model statusini yoqish/o''chirish'),
    ('/api/tech/gscode/upload', 'GS kodlarni modelga yuklash'),
    ('/api/tech/gscode/count', 'Model bo''yicha mavjud GS kod qoldig''ini ko''rish'),
    ('/api/tech/gscode/report', 'GS kod yuklash/ishlatish hisobotini ko''rish'),
    ('/api/tech/lines/add_gp_component', 'Liniya uchun GP komponent sozlamasini qo''shish'),
    ('/api/tech/printers/add', 'Liniya printerini qo''shish'),
    ('/api/tech/printers/delete', 'Liniya printerini o''chirish'),
    ('/api/tech/printers-v2/all', 'V2 printerlar ro''yxati'),
    ('/api/tech/printers-v2/by-line', 'Liniya bo''yicha V2 printerlar ro''yxati'),
    ('/api/tech/printers-v2/get', 'V2 printer ma''lumotini ko''rish'),
    ('/api/tech/printers-v2/add', 'V2 printer qo''shish'),
    ('/api/tech/printers-v2/delete', 'V2 printerni o''chirish'),
    ('/api/tech/printers-v2/local', 'Server kompyuteridagi o''rnatilgan printerlar ro''yxati'),
    ('/api/tech/printers-v2/jobs', 'Tanlangan V2 printerning joriy print queue ro''yxati'),
    ('/api/tech/printers-v2/detect-language', 'V2 printer uchun TSPL/ZPL/GDI tilini aniqlash'),
    ('/api/tech/printers-v2/update-language', 'V2 printer print tilini yangilash'),
    ('/api/tech/printers-v2/refresh-gdi', 'GDI print worker sozlamalarini yangilash'),
    ('/api/tech/printers-v2/test-print', 'V2 printerga test etiketka yuborish'),
    ('/api/tech/printers-v2/metrics', 'V2 print metrikalari (kunlik)'),
    ('/api/tech/printers-v2/metrics/errors', 'V2 print xatolari ro''yxati'),
    ('/api/tech/printers-v2/metrics/reset', 'V2 print eventlarini oralik bo''yicha tozalash'),
    ('/api/tech/label-templates/all', 'Etiketka shablonlari ro''yxati'),
    ('/api/tech/label-templates/get', 'Etiketka shablonini ko''rish'),
    ('/api/tech/label-templates/create', 'Yangi etiketka shablonini yaratish'),
    ('/api/tech/label-templates/duplicate', 'Mavjud etiketka shablonidan nusxa yaratish'),
    ('/api/tech/label-templates/update', 'Etiketka shablonini yangilash'),
    ('/api/tech/label-templates/delete', 'Etiketka shablonini o''chirish'),
    ('/api/tech/label-templates/preview', 'Printer V2 uchun etiketka oldindan ko''rish'),
    ('/api/tech/label-templates/image/upload', 'Etiketka shabloniga rasm yuklash'),
    ('/api/tech/brands/all', 'Model kartochkasidagi mavjud brandlar ro''yxati'),
    ('/api/tech/brand-logos/all', 'Brand logo bog''lanmalarini ko''rish'),
    ('/api/tech/brand-logos/upsert', 'Brand uchun logo biriktirish yoki yangilash'),
    ('/api/tech/brand-logos/delete', 'Brand logoni deaktiv qilish'),
    ('/api/production/info', 'Ishlab chiqarish umumiy ma''lumotlari va serial hisobotini ko''rish'),
    ('/api/production/report/xlsx', 'Ishlab chiqarish hisobotini XLSX faylga yaratish'),
    ('/api/production/components/all', 'Production komponentlar katalogini ko''rish'),
    ('/api/production/components/add', 'Production komponent qo''shish'),
    ('/api/production/components/update', 'Production komponent ma''lumotlarini yangilash'),
    ('/api/production/components/delete', 'Production komponentni o''chirish'),
    ('/api/production/components/upload', 'Production komponentlarni XLSX orqali import qilish'),
    ('/api/production/components/types', 'Production komponent turlarini ko''rish'),
    ('/api/production/components/units', 'Production komponent o''lchov birliklarini ko''rish'),
    ('/api/production/components/template', 'Production komponentlar XLSX shablonini yuklab olish'),
    ('/api/production/components/photo/upload', 'Production komponent rasmini yuklash'),
    ('/api/production/line_responsibles/all', 'Liniya mas''ullari ro''yxatini ko''rish'),
    ('/api/production/line_responsibles/add', 'Liniya mas''ulini tayinlash'),
    ('/api/production/line_responsibles/delete', 'Liniya mas''ulini olib tashlash'),
    ('/api/production/consumption-norm/models', 'Sarf normasi bo''yicha modellar ro''yxati'),
    ('/api/production/consumption-norm/items', 'Model uchun sarf normasi qatorlarini ko''rish'),
    ('/api/production/consumption-norm/lines', 'Sarf normasi uchun liniyalar ro''yxati'),
    ('/api/production/consumption-norm/update-lines', 'Sarf normasi qatorida liniyalarni yangilash'),
    ('/api/production/consumption-norm/add', 'Sarf normasiga qator qo''shish'),
    ('/api/production/consumption-norm/delete', 'Sarf normasi qatorini o''chirish'),
    ('/api/production/consumption-norm/upload', 'Sarf normasini XLSX orqali yuklash'),
    ('/api/production/consumption-norm/template', 'Sarf normasi XLSX shablonini yuklab olish'),
    ('/api/production/plan/day', 'Kunlik ishlab chiqarish rejasini ko''rish'),
    ('/api/production/plan/month', 'Oylik reja jadvali (Excel ko''rinishi)'),
    ('/api/production/plan/save', 'Reja kataklarini saqlash'),
    ('/api/production/plan/report', 'Reja va bajarilish hisoboti'),
    ('/api/production/plan/dashboard', 'Reja dashboard'),
    ('/api/production/plan/lock-day', 'Kunlik rejani tasdiqlash'),
    ('/api/production/plan/lock-month', 'Oylik rejani tasdiqlash'),
    ('/api/production/plan/unlock-day', 'Kunlik rejani qayta ochish (tahrirlash uchun)'),
    ('/api/production/plan/unlock-month', 'Oylik tasdiqlangan kunlarni qayta ochish'),
    ('/api/production/plan/allowed-models', 'Rejadagi modellar ro''yxati'),
    ('/api/production/plan/export', 'Reja Excel export'),
    ('/api/production/plan/import', 'Reja Excel import'),
    ('/api/production/shifts/get', 'Ishlab chiqarish smena vaqtlarini ko''rish'),
    ('/api/production/shifts/save', 'Ishlab chiqarish smena vaqtlarini saqlash'),
    ('/api/production/eshik/all', 'Eshik liniyasi modellar ro''yxati'),
    ('/api/production/eshik/add', 'Eshik liniyasiga model qo''shish'),
    ('/api/production/eshik/update', 'Eshik liniyasi modelini yangilash'),
    ('/api/production/eshik/delete', 'Eshik liniyasidan modelni o''chirish'),
    ('/api/lines/all', 'Liniyalar ro''yxatini ko''rish'),
    ('/api/lines/gp_component/all', 'Liniya GP komponentlari ro''yxatini ko''rish'),
    ('/api/lines/gp_component/delete', 'Liniya GP komponent sozlamasini o''chirish'),
    ('/api/lines/product/last', 'Liniya bo''yicha oxirgi komponent mahsulotlarini ko''rish'),
    ('/api/lines/product/add', 'Liniyaga komponent mahsulotini qo''shish va label chop etish'),
    ('/api/lines/printers/all', 'Liniya printerlari ro''yxatini ko''rish'),
    ('/api/lines/product/reprint', 'Liniya mahsulot labelini qayta chop etish'),
    ('/api/lines/balance', 'Liniya balans/qoldiq ma''lumotlarini ko''rish'),
    ('/api/lines/balance/adjust', 'Liniya komponent balansini qo''lda o''zgartirish'),
    ('/api/lines/balance/transactions', 'Liniya balans tranzaksiyalari tarixini ko''rish'),
    ('/api/lines/product/balance', 'Liniya mahsulot balansi (faol serial, WIP)'),
    ('/api/lines/product/report', 'Mahsulot o''tkazishlar hisoboti (transferred)'),
    ('/api/lines/last', 'Tanlangan liniyaning oxirgi mahsulotlarini ko''rish'),
    ('/api/lines/report', 'Liniyalar ishlab chiqarish hisobotini ko''rish va XLSX uchun ma''lumot olish'),
    ('/api/lines/auxiliary/report', 'Yordamchi liniyalar komponent balansi: kirim, chiqim, qoldiq'),
    ('/api/lines/dashboard', 'Dashboard uchun bugungi liniya natijalarini ko''rish'),
    ('/api/lines/reja', 'Bugungi ishlab chiqarish rejasini ko''rish'),
    ('/api/lines/reja/update', 'Bugungi ishlab chiqarish rejasini yangilash'),
    ('/api/lines/brigadir/items', 'Brigadir: liniya uchun tayyor komponentlar'),
    ('/api/lines/brigadir/confirm', 'Brigadir: komponent qabulini tasdiqlash'),
    ('/api/lines/brigadir/confirm-all', 'Brigadir: barcha pozitsiyalarni qabul qilish'),
    ('/api/lines/yigish/v2/print', 'Yi''g''ish liniyasi: serial yaratish va chop etish'),
    ('/api/lines/yigish/v2/reprint', 'Yi''g''ish liniyasi: serialni qayta chop etish'),
    ('/api/lines/eshik/v2/print', 'Eshik liniyasi: freeze+ref etiketka chop etish'),
    ('/api/lines/eshik/v2/reprint', 'Eshik liniyasi: serialni qayta chop etish'),
    ('/api/lines/eshik/v2/sessions/last', 'Eshik liniyasi: oxirgi chop etish sessiyalari'),
    ('/api/lines/qadoqlash/complete', 'Qadoqlash liniyasi: acc/eshik/product skan va barcha printerlarga chop'),
    ('/api/lines/qadoqlash/reprint', 'Qadoqlash liniyasi: tanlangan printer/shablonga bitta etiketka qayta chop'),
    ('/api/lines/qadoqlash/sessions/last', 'Qadoqlash liniyasi: oxirgi sessiyalar'),
    ('/api/ware/stock/all', 'Ombor qoldig''ini ko''rish'),
    ('/api/ware/stock/snapshot', 'Tanlangan sana uchun ombor qoldiq slepogi'),
    ('/api/ware/gp-products/last', 'Ombor GP mahsulotlarining oxirgi qabul qilinganlari'),
    ('/api/ware/gp-products/history', 'Ombor GP mahsulot qabul qilish tranzaksiya tarixi'),
    ('/api/ware/gp-products/report', 'Ombor GP mahsulot balansi va davr bo''yicha hisobot'),
    ('/api/ware/income', 'Omborga komponent kirimini qo''shish'),
    ('/api/ware/income/history', 'Ombor kirimlari tarixini ko''rish'),
    ('/api/ware/upload', 'Ombor kirimini Excel orqali yuklash'),
    ('/api/ware/request/build', 'Model bo''yicha ombor buyurtmasi ro''yxatini hisoblash'),
    ('/api/ware/request/confirm', 'Ombor buyurtmasi nakladnomasini yaratish'),
    ('/api/ware/delivery-notes', 'Ombor chiqim: nakladnomalar ro''yxati'),
    ('/api/ware/delivery-notes/items', 'Ombor chiqim: nakladnoma tarkibi'),
    ('/api/ware/delivery-notes/snapshots', 'Nakladnoya nusxalari ro''yxati'),
    ('/api/ware/delivery-notes/snapshots/items', 'Nakladnoya nusxasi tarkibi'),
    ('/api/ware/delivery-notes/items/update', 'Nakladnoma pozitsiyasi miqdorini yangilash'),
    ('/api/ware/delivery-notes/items/confirm', 'Nakladnoma pozitsiyasini tayyor deb belgilash'),
    ('/api/ware/delivery-notes/items/unconfirm', 'Nakladnoy pozitsiyasini tayyor emas qilish'),
    ('/api/ware/delivery-notes/items/confirm-all', 'Nakladnoma barcha pozitsiyalarini tasdiqlash'),
    ('/api/ware/delivery-notes/items/unconfirm-all', 'Nakladnoma barcha pozitsiyalarini tayyor emas qilish'),
    ('/api/ware/delivery-notes/items/delete', 'Nakladnoma pozitsiyasini o''chirish'),
    ('/api/writeoff/responsibles/all', 'Hisobdan chiqarish mas''ullari ro''yxati'),
    ('/api/writeoff/responsibles/add', 'Hisobdan chiqarish mas''ulini tayinlash'),
    ('/api/writeoff/responsibles/delete', 'Hisobdan chiqarish mas''ulini olib tashlash'),
    ('/api/writeoff/documents/list', 'Hisobdan chiqarish hujjatlari ro''yxati'),
    ('/api/writeoff/documents/get', 'Hisobdan chiqarish hujjati'),
    ('/api/writeoff/documents/create', 'Yangi hisobdan chiqarish hujjati'),
    ('/api/writeoff/documents/delete', 'Hisobdan chiqarish hujjatini o''chirish'),
    ('/api/writeoff/documents/items/save', 'Hisobdan chiqarish qatorlarini saqlash'),
    ('/api/writeoff/documents/submit', 'Hisobdan chiqarish hujjatini tasdiqlashga yuborish'),
    ('/api/writeoff/documents/approve', 'Hisobdan chiqarish hujjatini tasdiqlash'),
    ('/api/writeoff/documents/reject', 'Hisobdan chiqarish hujjatini rad etish'),
    ('/api/writeoff/documents/export', 'Hisobdan chiqarish Excel export'),
    ('/api/writeoff/documents/import', 'Hisobdan chiqarish Excel import'),
    ('/api/writeoff/documents/template', 'Hisobdan chiqarish Excel shablon'),
    ('/api/writeoff/records/list', 'Tasdiqlangan hisobdan chiqarish yozuvlari'),
    ('/api/writeoff/catalog/lines', 'Hisobdan chiqarish uchun liniyalar'),
    ('/api/writeoff/catalog/items', 'Hisobdan chiqarish uchun model/komponentlar'),
    ('/api/writeoff/is-responsible', 'Joriy foydalanuvchi hisobdan chiqarish mas''uli ekanligini tekshirish');

UPDATE auth.routes r
SET comment = t.comment,
    is_hidden = false
FROM tmp_route_comments t
WHERE r.route = t.route;

INSERT INTO auth.routes (route, comment, is_hidden)
SELECT t.route, t.comment, false
FROM tmp_route_comments t
WHERE NOT EXISTS (
    SELECT 1 FROM auth.routes r WHERE r.route = t.route
);

UPDATE auth.routes
SET is_hidden = true
WHERE NOT EXISTS (
    SELECT 1 FROM tmp_route_comments t WHERE t.route = auth.routes.route
);

WITH base_users AS (
    SELECT DISTINCT p.user_id
    FROM auth.permissions p
    INNER JOIN auth.routes r ON r.id = p.route_id
    WHERE r.route = '/api/production/components/all'
),
helper_routes AS (
    SELECT id
    FROM auth.routes
    WHERE is_hidden = false
      AND route IN (
        '/api/production/components/types',
        '/api/production/components/units',
        '/api/production/components/template',
        '/api/production/components/photo/upload',
        '/api/production/line_responsibles/all',
        '/api/production/line_responsibles/add',
        '/api/production/line_responsibles/delete'
      )
)
INSERT INTO auth.permissions (user_id, route_id)
SELECT base_users.user_id, helper_routes.id
FROM base_users
CROSS JOIN helper_routes
WHERE NOT EXISTS (
    SELECT 1
    FROM auth.permissions existing
    WHERE existing.user_id = base_users.user_id
      AND existing.route_id = helper_routes.id
);

WITH balance_users AS (
    SELECT DISTINCT p.user_id
    FROM auth.permissions p
    INNER JOIN auth.routes r ON r.id = p.route_id
    WHERE r.route = '/api/lines/balance'
),
balance_helper_routes AS (
    SELECT id
    FROM auth.routes
    WHERE is_hidden = false
      AND route IN (
        '/api/lines/balance/adjust',
        '/api/lines/balance/transactions',
        '/api/lines/product/balance',
        '/api/lines/product/report',
        '/api/ware/gp-products/last',
        '/api/ware/gp-products/history',
        '/api/ware/gp-products/report'
      )
)
INSERT INTO auth.permissions (user_id, route_id)
SELECT balance_users.user_id, balance_helper_routes.id
FROM balance_users
CROSS JOIN balance_helper_routes
WHERE NOT EXISTS (
    SELECT 1
    FROM auth.permissions existing
    WHERE existing.user_id = balance_users.user_id
      AND existing.route_id = balance_helper_routes.id
);

COMMIT;
