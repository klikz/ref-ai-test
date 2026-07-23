import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"

export default function RePrintPage() {
  return (
    <PageContainer
      title="Qayta chop"
      description="Liniya etiketkalarini qayta chop etish"
    >
      <Panel title="Liniyalar yo'q">
        <p className="text-sm text-muted-foreground">
          AC liniyalar (T1/T2/T3/Ichki/Fin Press/Radiator/Klapan) olib tashlandi.
          Yangi sovutgich liniyalari qo‘shilgach, qayta chop shu yerda ishlaydi.
        </p>
      </Panel>
    </PageContainer>
  )
}
