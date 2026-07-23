package routecomments

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type RouteComment struct {
	Route   string
	Comment string
}

func All() []RouteComment {
	return []RouteComment{
		// Users / permissions
		{Route: "/api/user/create", Comment: "Foydalanuvchi yaratish"},
		{Route: "/api/user/update", Comment: "Foydalanuvchi ma'lumotlari yoki parolini yangilash"},
		{Route: "/api/user/delete", Comment: "Foydalanuvchini deaktiv qilish"},
		{Route: "/api/user/getall", Comment: "Foydalanuvchilar ro'yxatini ko'rish"},
		{Route: "/api/user/:id", Comment: "Foydalanuvchi profili va ruxsatlarini ko'rish"},
		{Route: "/api/user/permission/:id", Comment: "Foydalanuvchi route ruxsatlarini o'zgartirish"},

		{Route: "/api/serial/info", Comment: "Serial/kompressor bo'yicha mahsulot, params va skan surati"},
		{Route: "/api/lab/info", Comment: "Laboratoriya (VTM) BxData — serial bo'yicha test natijalari"},

		// Technology / models and GS codes
		{Route: "/api/tech/models/add", Comment: "Modellarni Excel orqali qo'shish"},
		{Route: "/api/tech/models/all", Comment: "Modellar katalogini ko'rish"},
		{Route: "/api/tech/models/update", Comment: "Model ma'lumotlarini Excel orqali yangilash"},
		{Route: "/api/tech/models/status", Comment: "Model statusini yoqish/o'chirish"},
		{Route: "/api/tech/gscode/upload", Comment: "GS kodlarni modelga yuklash"},
		{Route: "/api/tech/gscode/count", Comment: "Model bo'yicha mavjud GS kod qoldig'ini ko'rish"},
		{Route: "/api/tech/gscode/report", Comment: "GS kod yuklash/ishlatish hisobotini ko'rish"},

		// Technology / production settings
		{Route: "/api/tech/lines/add_gp_component", Comment: "Liniya uchun GP komponent sozlamasini qo'shish"},
		{Route: "/api/tech/printers/add", Comment: "Liniya printerini qo'shish"},
		{Route: "/api/tech/printers/delete", Comment: "Liniya printerini o'chirish"},
		{Route: "/api/tech/printers-v2/all", Comment: "V2 printerlar ro'yxati"},
		{Route: "/api/tech/printers-v2/by-line", Comment: "Liniya bo'yicha V2 printerlar ro'yxati"},
		{Route: "/api/tech/printers-v2/get", Comment: "V2 printer ma'lumotini ko'rish"},
		{Route: "/api/tech/printers-v2/add", Comment: "V2 printer qo'shish"},
		{Route: "/api/tech/printers-v2/delete", Comment: "V2 printerni o'chirish"},
		{Route: "/api/tech/printers-v2/local", Comment: "Server kompyuteridagi o'rnatilgan printerlar ro'yxati"},
		{Route: "/api/tech/printers-v2/jobs", Comment: "Tanlangan V2 printerning joriy print queue ro'yxati"},
		{Route: "/api/tech/printers-v2/detect-language", Comment: "V2 printer uchun TSPL/ZPL/GDI tilini aniqlash"},
		{Route: "/api/tech/label-templates/all", Comment: "Etiketka shablonlari ro'yxati"},
		{Route: "/api/tech/label-templates/get", Comment: "Etiketka shablonini ko'rish"},
		{Route: "/api/tech/label-templates/create", Comment: "Yangi etiketka shablonini yaratish"},
		{Route: "/api/tech/label-templates/duplicate", Comment: "Mavjud etiketka shablonidan nusxa yaratish"},
		{Route: "/api/tech/label-templates/update", Comment: "Etiketka shablonini yangilash"},
		{Route: "/api/tech/label-templates/delete", Comment: "Etiketka shablonini o'chirish"},
		{Route: "/api/tech/label-templates/preview", Comment: "Printer V2 uchun etiketka oldindan ko'rish"},
		{Route: "/api/tech/label-templates/image/upload", Comment: "Etiketka shabloniga rasm yuklash"},
		{Route: "/api/tech/brands/all", Comment: "Model kartochkasidagi mavjud brandlar ro'yxati"},
		{Route: "/api/tech/brand-logos/all", Comment: "Brand logo bog'lanmalarini ko'rish"},
		{Route: "/api/tech/brand-logos/upsert", Comment: "Brand uchun logo biriktirish yoki yangilash"},
		{Route: "/api/tech/brand-logos/delete", Comment: "Brand logoni deaktiv qilish"},

		// Production
		{Route: "/api/production/info", Comment: "Ishlab chiqarish umumiy ma'lumotlari va serial hisobotini ko'rish"},
		{Route: "/api/production/report/xlsx", Comment: "Ishlab chiqarish hisobotini XLSX faylga yaratish"},
		{Route: "/api/production/components/all", Comment: "Production komponentlar katalogini ko'rish"},
		{Route: "/api/production/components/add", Comment: "Production komponent qo'shish"},
		{Route: "/api/production/components/update", Comment: "Production komponent ma'lumotlarini yangilash"},
		{Route: "/api/production/components/delete", Comment: "Production komponentni o'chirish"},
		{Route: "/api/production/components/upload", Comment: "Production komponentlarni XLSX orqali import qilish"},
		{Route: "/api/production/components/types", Comment: "Production komponent turlarini ko'rish"},
		{Route: "/api/production/components/units", Comment: "Production komponent o'lchov birliklarini ko'rish"},
		{Route: "/api/production/components/template", Comment: "Production komponentlar XLSX shablonini yuklab olish"},
		{Route: "/api/production/components/photo/upload", Comment: "Production komponent rasmini yuklash"},
		{Route: "/api/production/line_responsibles/all", Comment: "Liniya mas'ullari ro'yxatini ko'rish"},
		{Route: "/api/production/line_responsibles/add", Comment: "Liniya mas'ulini tayinlash"},
		{Route: "/api/production/line_responsibles/delete", Comment: "Liniya mas'ulini olib tashlash"},
		{Route: "/api/production/consumption-norm/models", Comment: "Sarf normasi bo'yicha modellar ro'yxati"},
		{Route: "/api/production/consumption-norm/items", Comment: "Model uchun sarf normasi qatorlarini ko'rish"},
		{Route: "/api/production/consumption-norm/lines", Comment: "Sarf normasi uchun liniyalar ro'yxati"},
		{Route: "/api/production/consumption-norm/update-lines", Comment: "Sarf normasi qatorida liniyalarni yangilash"},
		{Route: "/api/production/consumption-norm/add", Comment: "Sarf normasiga qator qo'shish"},
		{Route: "/api/production/consumption-norm/delete", Comment: "Sarf normasi qatorini o'chirish"},
		{Route: "/api/production/consumption-norm/upload", Comment: "Sarf normasini XLSX orqali yuklash"},
		{Route: "/api/production/consumption-norm/template", Comment: "Sarf normasi XLSX shablonini yuklab olish"},
		{Route: "/api/production/plan/day", Comment: "Kunlik ishlab chiqarish rejasini ko'rish"},
		{Route: "/api/production/plan/month", Comment: "Oylik reja jadvali (Excel ko'rinishi)"},
		{Route: "/api/production/plan/save", Comment: "Reja kataklarini saqlash"},
		{Route: "/api/production/plan/report", Comment: "Reja va bajarilish hisoboti"},
		{Route: "/api/production/plan/dashboard", Comment: "Reja dashboard"},
		{Route: "/api/production/plan/lock-day", Comment: "Kunlik rejani tasdiqlash"},
		{Route: "/api/production/plan/lock-month", Comment: "Oylik rejani tasdiqlash"},
		{Route: "/api/production/plan/unlock-day", Comment: "Kunlik rejani qayta ochish (tahrirlash uchun)"},
		{Route: "/api/production/plan/unlock-month", Comment: "Oylik tasdiqlangan kunlarni qayta ochish"},
		{Route: "/api/production/plan/allowed-models", Comment: "Rejadagi modellar ro'yxati"},
		{Route: "/api/production/plan/export", Comment: "Reja Excel export"},
		{Route: "/api/production/plan/import", Comment: "Reja Excel import"},
		{Route: "/api/production/shifts/get", Comment: "Ishlab chiqarish smena vaqtlarini ko'rish"},
		{Route: "/api/production/shifts/save", Comment: "Ishlab chiqarish smena vaqtlarini saqlash"},
		{Route: "/api/production/eshik/all", Comment: "Eshik liniyasi komponentlar ro'yxati"},
		{Route: "/api/production/eshik/add", Comment: "Eshik liniyasiga komponent qo'shish"},
		{Route: "/api/production/eshik/update", Comment: "Eshik liniyasi komponentini yangilash"},
		{Route: "/api/production/eshik/delete", Comment: "Eshik liniyasidan komponentni o'chirish"},

		// Lines
		{Route: "/api/lines/all", Comment: "Liniyalar ro'yxatini ko'rish"},
		{Route: "/api/lines/gp_component/all", Comment: "Liniya GP komponentlari ro'yxatini ko'rish"},
		{Route: "/api/lines/gp_component/delete", Comment: "Liniya GP komponent sozlamasini o'chirish"},
		{Route: "/api/lines/product/last", Comment: "Liniya bo'yicha oxirgi komponent mahsulotlarini ko'rish"},
		{Route: "/api/lines/product/add", Comment: "Liniyaga komponent mahsulotini qo'shish va label chop etish"},
		{Route: "/api/lines/printers/all", Comment: "Liniya printerlari ro'yxatini ko'rish"},
		{Route: "/api/lines/product/reprint", Comment: "Liniya mahsulot labelini qayta chop etish"},
		{Route: "/api/lines/balance", Comment: "Liniya balans/qoldiq ma'lumotlarini ko'rish"},
		{Route: "/api/lines/balance/adjust", Comment: "Liniya komponent balansini qo'lda o'zgartirish"},
		{Route: "/api/lines/balance/transactions", Comment: "Liniya balans tranzaksiyalari tarixini ko'rish"},
		{Route: "/api/lines/product/balance", Comment: "Liniya mahsulot balansi (faol serial, WIP)"},
		{Route: "/api/lines/product/report", Comment: "Mahsulot o'tkazishlar hisoboti (transferred)"},
		{Route: "/api/lines/last", Comment: "Tanlangan liniyaning oxirgi mahsulotlarini ko'rish"},
		{Route: "/api/lines/report", Comment: "Liniyalar ishlab chiqarish hisobotini ko'rish va XLSX uchun ma'lumot olish"},
		{Route: "/api/lines/auxiliary/report", Comment: "Yordamchi liniyalar komponent balansi: kirim, chiqim, qoldiq"},
		{Route: "/api/lines/dashboard", Comment: "Dashboard uchun bugungi liniya natijalarini ko'rish"},
		{Route: "/api/lines/reja", Comment: "Bugungi ishlab chiqarish rejasini ko'rish"},
		{Route: "/api/lines/reja/update", Comment: "Bugungi ishlab chiqarish rejasini yangilash"},
		{Route: "/api/lines/brigadir/items", Comment: "Brigadir: liniya uchun tayyor komponentlar"},
		{Route: "/api/lines/brigadir/confirm", Comment: "Brigadir: komponent qabulini tasdiqlash"},
		{Route: "/api/lines/brigadir/confirm-all", Comment: "Brigadir: barcha pozitsiyalarni qabul qilish"},
		{Route: "/api/lines/yigish/v2/print", Comment: "Yi'g'ish liniyasi: serial yaratish va chop etish"},
		{Route: "/api/lines/yigish/v2/reprint", Comment: "Yi'g'ish liniyasi: serialni qayta chop etish"},
		{Route: "/api/lines/eshik/v2/print", Comment: "Eshik liniyasi: serial yaratish va chop etish"},
		{Route: "/api/lines/eshik/v2/reprint", Comment: "Eshik liniyasi: serialni qayta chop etish"},
		{Route: "/api/lines/eshik/v2/sessions/last", Comment: "Eshik liniyasi: oxirgi chop etish sessiyalari"},

		// Warehouse
		{Route: "/api/ware/stock/all", Comment: "Ombor qoldig'ini ko'rish"},
		{Route: "/api/ware/stock/snapshot", Comment: "Tanlangan sana uchun ombor qoldiq slepogi"},
		{Route: "/api/ware/gp-products/last", Comment: "Ombor GP mahsulotlarining oxirgi qabul qilinganlari"},
		{Route: "/api/ware/gp-products/history", Comment: "Ombor GP mahsulot qabul qilish tranzaksiya tarixi"},
		{Route: "/api/ware/gp-products/report", Comment: "Ombor GP mahsulot balansi va davr bo'yicha hisobot"},
		{Route: "/api/ware/income", Comment: "Omborga komponent kirimini qo'shish"},
		{Route: "/api/ware/income/history", Comment: "Ombor kirimlari tarixini ko'rish"},
		{Route: "/api/ware/upload", Comment: "Ombor kirimini Excel orqali yuklash"},
		{Route: "/api/ware/request/build", Comment: "Model bo'yicha ombor buyurtmasi ro'yxatini hisoblash"},
		{Route: "/api/ware/request/confirm", Comment: "Ombor buyurtmasi nakladnomasini yaratish"},
		{Route: "/api/ware/delivery-notes", Comment: "Ombor chiqim: nakladnomalar ro'yxati"},
		{Route: "/api/ware/delivery-notes/items", Comment: "Ombor chiqim: nakladnoma tarkibi"},
		{Route: "/api/ware/delivery-notes/snapshots", Comment: "Nakladnoya nusxalari ro'yxati"},
		{Route: "/api/ware/delivery-notes/snapshots/items", Comment: "Nakladnoya nusxasi tarkibi"},
		{Route: "/api/ware/delivery-notes/items/update", Comment: "Nakladnoma pozitsiyasi miqdorini yangilash"},
		{Route: "/api/ware/delivery-notes/items/confirm", Comment: "Nakladnoma pozitsiyasini tayyor deb belgilash"},
		{Route: "/api/ware/delivery-notes/items/unconfirm", Comment: "Nakladnoy pozitsiyasini tayyor emas qilish"},
		{Route: "/api/ware/delivery-notes/items/confirm-all", Comment: "Nakladnoma barcha pozitsiyalarini tasdiqlash"},
		{Route: "/api/ware/delivery-notes/items/unconfirm-all", Comment: "Nakladnoma barcha pozitsiyalarini tayyor emas qilish"},
		{Route: "/api/ware/delivery-notes/items/delete", Comment: "Nakladnoma pozitsiyasini o'chirish"},

		// Write-off
		{Route: "/api/writeoff/responsibles/all", Comment: "Hisobdan chiqarish mas'ullari ro'yxati"},
		{Route: "/api/writeoff/responsibles/add", Comment: "Hisobdan chiqarish mas'ulini tayinlash"},
		{Route: "/api/writeoff/responsibles/delete", Comment: "Hisobdan chiqarish mas'ulini olib tashlash"},
		{Route: "/api/writeoff/documents/list", Comment: "Hisobdan chiqarish hujjatlari ro'yxati"},
		{Route: "/api/writeoff/documents/get", Comment: "Hisobdan chiqarish hujjati"},
		{Route: "/api/writeoff/documents/create", Comment: "Yangi hisobdan chiqarish hujjati"},
		{Route: "/api/writeoff/documents/delete", Comment: "Hisobdan chiqarish hujjatini o'chirish"},
		{Route: "/api/writeoff/documents/items/save", Comment: "Hisobdan chiqarish qatorlarini saqlash"},
		{Route: "/api/writeoff/documents/submit", Comment: "Hisobdan chiqarish hujjatini tasdiqlashga yuborish"},
		{Route: "/api/writeoff/documents/approve", Comment: "Hisobdan chiqarish hujjatini tasdiqlash"},
		{Route: "/api/writeoff/documents/reject", Comment: "Hisobdan chiqarish hujjatini rad etish"},
		{Route: "/api/writeoff/documents/export", Comment: "Hisobdan chiqarish Excel export"},
		{Route: "/api/writeoff/documents/import", Comment: "Hisobdan chiqarish Excel import"},
		{Route: "/api/writeoff/documents/template", Comment: "Hisobdan chiqarish Excel shablon"},
		{Route: "/api/writeoff/records/list", Comment: "Tasdiqlangan hisobdan chiqarish yozuvlari"},
		{Route: "/api/writeoff/catalog/lines", Comment: "Hisobdan chiqarish uchun liniyalar"},
		{Route: "/api/writeoff/catalog/items", Comment: "Hisobdan chiqarish uchun model/komponentlar"},
		{Route: "/api/writeoff/is-responsible", Comment: "Joriy foydalanuvchi hisobdan chiqarish mas'uli ekanligini tekshirish"},
	}
}

func Apply(ctx context.Context, db *sql.DB, dryRun bool) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	active := All()
	count := 0
	activeRoutes := make([]string, 0, len(active))

	for _, item := range active {
		if item.Route == "" {
			continue
		}
		count++
		activeRoutes = append(activeRoutes, item.Route)
		if dryRun {
			fmt.Printf("DRY-RUN upsert auth.routes route=%q comment=%q is_hidden=false\n", item.Route, item.Comment)
			continue
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE auth.routes
			SET comment = $2, is_hidden = false
			WHERE route = $1`, item.Route, item.Comment)
		if err != nil {
			return count, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return count, err
		}
		if affected == 0 {
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO auth.routes (route, comment, is_hidden) VALUES ($1, $2, false)`,
				item.Route,
				item.Comment,
			); err != nil {
				return count, err
			}
		}
	}

	if dryRun {
		fmt.Println("DRY-RUN hide routes not present in routecomments.All()")
	} else {
		if _, err := tx.ExecContext(ctx, `
			UPDATE auth.routes
			SET is_hidden = true
			WHERE NOT (route = ANY($1::text[]))`, pq.Array(activeRoutes)); err != nil {
			return count, err
		}
	}

	if dryRun {
		fmt.Println("DRY-RUN grant helper production component routes to users with components/all permission")
	} else if _, err := tx.ExecContext(ctx, `
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
		)`,
	); err != nil {
		return count, err
	}

	if dryRun {
		return count, nil
	}
	if err := tx.Commit(); err != nil {
		return count, err
	}
	return count, nil
}
