import { lazy, Suspense } from "react"
import { Routes, Route, Navigate } from "react-router-dom"
import { Global_Data } from "./config/config"
import MainLayout from "./layouts/MainLayout"
import { PageLoader } from "@/components/layout/page-loader"

const LoginPage = lazy(() => import("./pages/login"))
const Home = lazy(() => import("./pages/home"))
const UsersPage = lazy(() => import("./pages/users"))
const UserPage = lazy(() => import("./pages/user_id"))
const ModelsPage = lazy(() => import("./pages/models"))
const ProductionComponentsPage = lazy(() => import("./pages/production_components"))
const ConsumptionNormPage = lazy(() => import("./pages/consumption_norm"))
const ConsumptionNormIdPage = lazy(() => import("./pages/consumption_norm_id"))
const LineBalancePage = lazy(() => import("./pages/line_balance"))
const LineBalanceTransactionsPage = lazy(() => import("./pages/line_balance_transactions"))
const ProductBalancePage = lazy(() => import("./pages/product_balance"))
const ProductBalanceReportPage = lazy(() => import("./pages/product_balance_report"))
const LineResponsiblesPage = lazy(() => import("./pages/line_responsibles"))
const WareIncomePage = lazy(() => import("./pages/ware_income"))
const WareIncomeReportPage = lazy(() => import("./pages/ware_income_report"))
const WareRequestPage = lazy(() => import("./pages/ware_request"))
const WareRequestHistoryPage = lazy(() => import("./pages/ware_request_history"))
const WareOutcomePage = lazy(() => import("./pages/ware_outcome"))
const WareOutcomeDetailPage = lazy(() => import("./pages/ware_outcome_detail"))
const WareStockSnapshotPage = lazy(() => import("./pages/ware_stock_snapshot"))
const GsCodePgae = lazy(() => import("./pages/gscode"))
const PrintersV2Page = lazy(() => import("./pages/printers_v2"))
const PrintersV2MetricsPage = lazy(() => import("./pages/printers_v2_metrics"))
const LabelTemplatesPage = lazy(() => import("./pages/label-templates"))
const LabelTemplateEditorPage = lazy(() => import("./pages/label-template-editor"))
const ModelsIdPage = lazy(() => import("./pages/model_id"))
const RePrintPage = lazy(() => import("./pages/re_print"))
const LinesReport = lazy(() => import("./pages/lines_report"))
const WareIncomeReportDetailPage = lazy(() => import("./pages/ware_income_report_detail"))
const GsCodesReport = lazy(() => import("./pages/gscode_report"))
const Dashboard = lazy(() => import("./pages/dashboard"))
const ProductionPlanPage = lazy(() => import("./pages/production_plan"))
const ProductionPlanDashboardPage = lazy(() => import("./pages/production_plan_dashboard"))
const ProductionPlanReportPage = lazy(() => import("./pages/production_plan_report"))
const ProductionShiftsPage = lazy(() => import("./pages/production_shifts"))
const GPProductsReportPage = lazy(() => import("./pages/gp_products_report"))
const RejaPage = lazy(() => import("./pages/reja"))
const BrigadirPage = lazy(() => import("./pages/brigadir"))
const WriteoffPage = lazy(() => import("./pages/writeoff"))
const WriteoffIdPage = lazy(() => import("./pages/writeoff_id"))
const WriteoffApprovePage = lazy(() => import("./pages/writeoff_approve"))
const WriteoffRecordsPage = lazy(() => import("./pages/writeoff_records"))
const WriteoffResponsiblesPage = lazy(() => import("./pages/writeoff_responsibles"))
const TestModelsPage = lazy(() => import("./pages/test_models"))
const SerialInfoPage = lazy(() => import("./pages/serial_info"))
const LabPage = lazy(() => import("./pages/lab"))
const YigishPage = lazy(() => import("./pages/yigish"))
const EshikPage = lazy(() => import("./pages/eshik"))
const EshikProductionPage = lazy(() => import("./pages/eshik_production"))
const QadoqlashPage = lazy(() => import("./pages/qadoqlash"))
const MeasurementUnitsPage = lazy(() => import("./pages/measurement_units"))
const CameraTestPage = lazy(() => import("./pages/camera_test"))
const AgentPage = lazy(() => import("./pages/agent"))

export function App() {
  Global_Data.loadLocalData()
  const isLogged = Global_Data.isAuth()

  return (
    <Suspense fallback={<PageLoader />}>
      {!isLogged ? (
        <Routes>
          <Route path="/" element={<Navigate to="/login" />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
      ) : (
        <Routes>
          <Route path="/dashboard" element={<Dashboard />} />
          <Route element={<MainLayout />}>
            <Route path="/" element={<Navigate to="/home" />} />
            <Route index path="/home" element={<Home />} />
            <Route path="/serial-info" element={<SerialInfoPage />} />
            <Route path="/lab" element={<LabPage />} />
            <Route path="/users" element={<UsersPage />} />
            <Route path="/user/:id" element={<UserPage />} />
            <Route path="/models" element={<ModelsPage />} />
            <Route path="/production/components" element={<ProductionComponentsPage />} />
            <Route path="/components" element={<Navigate to="/production/components" replace />} />
            <Route path="/production/consumption-norm" element={<ConsumptionNormPage />} />
            <Route path="/production/consumption-norm/:id" element={<ConsumptionNormIdPage />} />
            <Route path="/production/balance" element={<LineBalancePage />} />
            <Route path="/production/balance/transactions" element={<LineBalanceTransactionsPage />} />
            <Route path="/production/product-balance" element={<ProductBalancePage />} />
            <Route path="/production/product-balance/report" element={<ProductBalanceReportPage />} />
            <Route path="/production/line-responsibles" element={<Navigate to="/tools/line-responsibles" replace />} />
            <Route path="/tools/line-responsibles" element={<LineResponsiblesPage />} />
            <Route path="/tools/writeoff-responsibles" element={<WriteoffResponsiblesPage />} />
            <Route path="/tools/measurement-units" element={<MeasurementUnitsPage />} />
            <Route path="/tools/camera-test" element={<CameraTestPage />} />
            <Route path="/agent" element={<AgentPage />} />
            <Route path="/ombor/kirim" element={<WareIncomePage />} />
            <Route path="/ombor/buyurtma" element={<WareRequestPage />} />
            <Route path="/ombor/buyurtma/tarix" element={<WareRequestHistoryPage />} />
            <Route path="/ombor/chiqim" element={<WareOutcomePage />} />
            <Route path="/ombor/chiqim/detail" element={<WareOutcomeDetailPage />} />
            <Route path="/ombor/kirim/report" element={<WareIncomeReportPage />} />
            <Route path="/ombor/kirim/report/detail" element={<WareIncomeReportDetailPage />} />
            <Route path="/ombor/snapshot" element={<WareStockSnapshotPage />} />
            <Route path="/models/:id" element={<ModelsIdPage />} />
            <Route path="/gscode" element={<GsCodePgae />} />
            <Route path="/gscode/report" element={<GsCodesReport />} />
            <Route path="/printers" element={<Navigate to="/printers-v2" replace />} />
            <Route path="/printers-v2" element={<PrintersV2Page />} />
            <Route path="/printers-v2/metrics" element={<PrintersV2MetricsPage />} />
            <Route path="/label-templates" element={<LabelTemplatesPage />} />
            <Route path="/label-templates/:id" element={<LabelTemplateEditorPage />} />
            <Route path="/brigadir" element={<BrigadirPage />} />
            <Route path="/yigish" element={<YigishPage />} />
            <Route path="/eshik" element={<EshikPage />} />
            <Route path="/qadoqlash" element={<QadoqlashPage />} />
            <Route path="/production/eshik" element={<EshikProductionPage />} />
            <Route path="/writeoff" element={<WriteoffPage />} />
            <Route path="/writeoff/responsibles" element={<Navigate to="/tools/writeoff-responsibles" replace />} />
            <Route path="/writeoff/approve" element={<WriteoffApprovePage />} />
            <Route path="/writeoff/records" element={<WriteoffRecordsPage />} />
            <Route path="/writeoff/:id" element={<WriteoffIdPage />} />
            <Route path="/reprint" element={<RePrintPage />} />
            <Route path="/report" element={<LinesReport />} />
            <Route path="/production/plan" element={<ProductionPlanPage />} />
            <Route path="/production/plan/dashboard" element={<ProductionPlanDashboardPage />} />
            <Route path="/production/plan/report" element={<ProductionPlanReportPage />} />
            <Route path="/production/shifts" element={<ProductionShiftsPage />} />
            <Route path="/report/gp-products" element={<GPProductsReportPage />} />
            <Route path="/reja" element={<RejaPage />} />
            <Route path="/test/models" element={<TestModelsPage />} />
            <Route path="*" element={<Navigate to="/home" />} />
          </Route>
        </Routes>
      )}
    </Suspense>
  )
}

export default App
