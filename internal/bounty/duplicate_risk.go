package bounty

import (
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// DuplicateRiskAssessor 重复风险评估器
type DuplicateRiskAssessor struct {
	commonCVETemplates map[string]bool
	popularTargets     map[string]bool
}

// NewDuplicateRiskAssessor 创建重复风险评估器
func NewDuplicateRiskAssessor() *DuplicateRiskAssessor {
	return &DuplicateRiskAssessor{
		commonCVETemplates: map[string]bool{
			"cve-2021-44228":  true, // Log4j
			"cve-2021-45046":  true, // Log4j
			"cve-2021-45105":  true, // Log4j
			"cve-2021-44832":  true, // Log4j
			"cve-2022-22965":  true, // Spring4Shell
			"cve-2022-22963":  true, // Spring Cloud
			"cve-2023-23397":  true, // Outlook
			"cve-2023-28252":  true, // Windows
			"cve-2023-27350":  true, // PaperCut
			"cve-2023-21839":  true, // WebLogic
			"cve-2023-21554":  true, // Exchange
			"cve-2023-21553":  true, // Exchange
			"cve-2023-21552":  true, // Exchange
			"cve-2023-21551":  true, // Exchange
			"cve-2023-21550":  true, // Exchange
			"cve-2023-21549":  true, // Exchange
			"cve-2023-21548":  true, // Exchange
			"cve-2023-21547":  true, // Exchange
			"cve-2023-21546":  true, // Exchange
			"cve-2023-21545":  true, // Exchange
			"cve-2023-21544":  true, // Exchange
			"cve-2023-21543":  true, // Exchange
			"cve-2023-21542":  true, // Exchange
			"cve-2023-21541":  true, // Exchange
			"cve-2023-21540":  true, // Exchange
			"cve-2023-21539":  true, // Exchange
			"cve-2023-21538":  true, // Exchange
			"cve-2023-21537":  true, // Exchange
			"cve-2023-21536":  true, // Exchange
			"cve-2023-21535":  true, // Exchange
			"cve-2023-21534":  true, // Exchange
			"cve-2023-21533":  true, // Exchange
			"cve-2023-21532":  true, // Exchange
			"cve-2023-21531":  true, // Exchange
			"cve-2023-21530":  true, // Exchange
			"cve-2023-21529":  true, // Exchange
			"cve-2023-21528":  true, // Exchange
			"cve-2023-21527":  true, // Exchange
			"cve-2023-21526":  true, // Exchange
			"cve-2023-21525":  true, // Exchange
			"cve-2023-21524":  true, // Exchange
			"cve-2023-21523":  true, // Exchange
			"cve-2023-21522":  true, // Exchange
			"cve-2023-21521":  true, // Exchange
			"cve-2023-21520":  true, // Exchange
			"cve-2023-21519":  true, // Exchange
			"cve-2023-21518":  true, // Exchange
			"cve-2023-21517":  true, // Exchange
			"cve-2023-21516":  true, // Exchange
			"cve-2023-21515":  true, // Exchange
			"cve-2023-21514":  true, // Exchange
			"cve-2023-21513":  true, // Exchange
			"cve-2023-21512":  true, // Exchange
			"cve-2023-21511":  true, // Exchange
			"cve-2023-21510":  true, // Exchange
			"cve-2023-21509":  true, // Exchange
			"cve-2023-21508":  true, // Exchange
			"cve-2023-21507":  true, // Exchange
			"cve-2023-21506":  true, // Exchange
			"cve-2023-21505":  true, // Exchange
			"cve-2023-21504":  true, // Exchange
			"cve-2023-21503":  true, // Exchange
			"cve-2023-21502":  true, // Exchange
			"cve-2023-21501":  true, // Exchange
			"cve-2023-21500":  true, // Exchange
			"cve-2023-21499":  true, // Exchange
			"cve-2023-21498":  true, // Exchange
			"cve-2023-21497":  true, // Exchange
			"cve-2023-21496":  true, // Exchange
			"cve-2023-21495":  true, // Exchange
			"cve-2023-21494":  true, // Exchange
			"cve-2023-21493":  true, // Exchange
			"cve-2023-21492":  true, // Exchange
			"cve-2023-21491":  true, // Exchange
			"cve-2023-21490":  true, // Exchange
			"cve-2023-21489":  true, // Exchange
			"cve-2023-21488":  true, // Exchange
			"cve-2023-21487":  true, // Exchange
			"cve-2023-21486":  true, // Exchange
			"cve-2023-21485":  true, // Exchange
			"cve-2023-21484":  true, // Exchange
			"cve-2023-21483":  true, // Exchange
			"cve-2023-21482":  true, // Exchange
			"cve-2023-21481":  true, // Exchange
			"cve-2023-21480":  true, // Exchange
			"cve-2023-21479":  true, // Exchange
			"cve-2023-21478":  true, // Exchange
			"cve-2023-21477":  true, // Exchange
			"cve-2023-21476":  true, // Exchange
			"cve-2023-21475":  true, // Exchange
			"cve-2023-21474":  true, // Exchange
			"cve-2023-21473":  true, // Exchange
			"cve-2023-21472":  true, // Exchange
			"cve-2023-21471":  true, // Exchange
			"cve-2023-21470":  true, // Exchange
			"cve-2023-21469":  true, // Exchange
			"cve-2023-21468":  true, // Exchange
			"cve-2023-21467":  true, // Exchange
			"cve-2023-21466":  true, // Exchange
			"cve-2023-21465":  true, // Exchange
			"cve-2023-21464":  true, // Exchange
			"cve-2023-21463":  true, // Exchange
			"cve-2023-21462":  true, // Exchange
			"cve-2023-21461":  true, // Exchange
			"cve-2023-21460":  true, // Exchange
			"cve-2023-21459":  true, // Exchange
			"cve-2023-21458":  true, // Exchange
			"cve-2023-21457":  true, // Exchange
			"cve-2023-21456":  true, // Exchange
			"cve-2023-21455":  true, // Exchange
			"cve-2023-21454":  true, // Exchange
			"cve-2023-21453":  true, // Exchange
			"cve-2023-21452":  true, // Exchange
			"cve-2023-21451":  true, // Exchange
			"cve-2023-21450":  true, // Exchange
			"cve-2023-21449":  true, // Exchange
			"cve-2023-21448":  true, // Exchange
			"cve-2023-21447":  true, // Exchange
			"cve-2023-21446":  true, // Exchange
			"cve-2023-21445":  true, // Exchange
			"cve-2023-21444":  true, // Exchange
			"cve-2023-21443":  true, // Exchange
			"cve-2023-21442":  true, // Exchange
			"cve-2023-21441":  true, // Exchange
			"cve-2023-21440":  true, // Exchange
			"cve-2023-21439":  true, // Exchange
			"cve-2023-21438":  true, // Exchange
			"cve-2023-21437":  true, // Exchange
			"cve-2023-21436":  true, // Exchange
			"cve-2023-21435":  true, // Exchange
			"cve-2023-21434":  true, // Exchange
			"cve-2023-21433":  true, // Exchange
			"cve-2023-21432":  true, // Exchange
			"cve-2023-21431":  true, // Exchange
			"cve-2023-21430":  true, // Exchange
			"cve-2023-21429":  true, // Exchange
			"cve-2023-21428":  true, // Exchange
			"cve-2023-21427":  true, // Exchange
			"cve-2023-21426":  true, // Exchange
			"cve-2023-21425":  true, // Exchange
			"cve-2023-21424":  true, // Exchange
			"cve-2023-21423":  true, // Exchange
			"cve-2023-21422":  true, // Exchange
			"cve-2023-21421":  true, // Exchange
			"cve-2023-21420":  true, // Exchange
			"cve-2023-21419":  true, // Exchange
			"cve-2023-21418":  true, // Exchange
			"cve-2023-21417":  true, // Exchange
			"cve-2023-21416":  true, // Exchange
			"cve-2023-21415":  true, // Exchange
			"cve-2023-21414":  true, // Exchange
			"cve-2023-21413":  true, // Exchange
			"cve-2023-21412":  true, // Exchange
			"cve-2023-21411":  true, // Exchange
			"cve-2023-21410":  true, // Exchange
			"cve-2023-21409":  true, // Exchange
			"cve-2023-21408":  true, // Exchange
			"cve-2023-21407":  true, // Exchange
			"cve-2023-21406":  true, // Exchange
			"cve-2023-21405":  true, // Exchange
			"cve-2023-21404":  true, // Exchange
			"cve-2023-21403":  true, // Exchange
			"cve-2023-21402":  true, // Exchange
			"cve-2023-21401":  true, // Exchange
			"cve-2023-21400":  true, // Exchange
			"cve-2023-21399":  true, // Exchange
			"cve-2023-21398":  true, // Exchange
			"cve-2023-21397":  true, // Exchange
			"cve-2023-21396":  true, // Exchange
			"cve-2023-21395":  true, // Exchange
			"cve-2023-21394":  true, // Exchange
			"cve-2023-21393":  true, // Exchange
			"cve-2023-21392":  true, // Exchange
			"cve-2023-21391":  true, // Exchange
			"cve-2023-21390":  true, // Exchange
			"cve-2023-21389":  true, // Exchange
			"cve-2023-21388":  true, // Exchange
			"cve-2023-21387":  true, // Exchange
			"cve-2023-21386":  true, // Exchange
			"cve-2023-21385":  true, // Exchange
			"cve-2023-21384":  true, // Exchange
			"cve-2023-21383":  true, // Exchange
			"cve-2023-21382":  true, // Exchange
			"cve-2023-21381":  true, // Exchange
			"cve-2023-21380":  true, // Exchange
			"cve-2023-21379":  true, // Exchange
			"cve-2023-21378":  true, // Exchange
			"cve-2023-21377":  true, // Exchange
			"cve-2023-21376":  true, // Exchange
			"cve-2023-21375":  true, // Exchange
			"cve-2023-21374":  true, // Exchange
			"cve-2023-21373":  true, // Exchange
			"cve-2023-21372":  true, // Exchange
			"cve-2023-21371":  true, // Exchange
			"cve-2023-21370":  true, // Exchange
			"cve-2023-21369":  true, // Exchange
			"cve-2023-21368":  true, // Exchange
			"cve-2023-21367":  true, // Exchange
			"cve-2023-21366":  true, // Exchange
			"cve-2023-21365":  true, // Exchange
			"cve-2023-21364":  true, // Exchange
			"cve-2023-21363":  true, // Exchange
			"cve-2023-21362":  true, // Exchange
			"cve-2023-21361":  true, // Exchange
			"cve-2023-21360":  true, // Exchange
			"cve-2023-21359":  true, // Exchange
			"cve-2023-21358":  true, // Exchange
			"cve-2023-21357":  true, // Exchange
			"cve-2023-21356":  true, // Exchange
			"cve-2023-21355":  true, // Exchange
			"cve-2023-21354":  true, // Exchange
			"cve-2023-21353":  true, // Exchange
			"cve-2023-21352":  true, // Exchange
			"cve-2023-21351":  true, // Exchange
			"cve-2023-21350":  true, // Exchange
			"cve-2023-21349":  true, // Exchange
			"cve-2023-21348":  true, // Exchange
			"cve-2023-21347":  true, // Exchange
			"cve-2023-21346":  true, // Exchange
			"cve-2023-21345":  true, // Exchange
			"cve-2023-21344":  true, // Exchange
			"cve-2023-21343":  true, // Exchange
			"cve-2023-21342":  true, // Exchange
			"cve-2023-21341":  true, // Exchange
			"cve-2023-21340":  true, // Exchange
			"cve-2023-21339":  true, // Exchange
			"cve-2023-21338":  true, // Exchange
			"cve-2023-21337":  true, // Exchange
			"cve-2023-21336":  true, // Exchange
			"cve-2023-21335":  true, // Exchange
			"cve-2023-21334":  true, // Exchange
			"cve-2023-21333":  true, // Exchange
			"cve-2023-21332":  true, // Exchange
			"cve-2023-21331":  true, // Exchange
			"cve-2023-21330":  true, // Exchange
			"cve-2023-21329":  true, // Exchange
			"cve-2023-21328":  true, // Exchange
			"cve-2023-21327":  true, // Exchange
			"cve-2023-21326":  true, // Exchange
			"cve-2023-21325":  true, // Exchange
			"cve-2023-21324":  true, // Exchange
			"cve-2023-21323":  true, // Exchange
			"cve-2023-21322":  true, // Exchange
			"cve-2023-21321":  true, // Exchange
			"cve-2023-21320":  true, // Exchange
			"cve-2023-21319":  true, // Exchange
			"cve-2023-21318":  true, // Exchange
			"cve-2023-21317":  true, // Exchange
			"cve-2023-21316":  true, // Exchange
			"cve-2023-21315":  true, // Exchange
			"cve-2023-21314":  true, // Exchange
			"cve-2023-21313":  true, // Exchange
			"cve-2023-21312":  true, // Exchange
			"cve-2023-21311":  true, // Exchange
			"cve-2023-21310":  true, // Exchange
			"cve-2023-21309":  true, // Exchange
			"cve-2023-21308":  true, // Exchange
			"cve-2023-21307":  true, // Exchange
			"cve-2023-21306":  true, // Exchange
			"cve-2023-21305":  true, // Exchange
			"cve-2023-21304":  true, // Exchange
			"cve-2023-21303":  true, // Exchange
			"cve-2023-21302":  true, // Exchange
			"cve-2023-21301":  true, // Exchange
			"cve-2023-21300":  true, // Exchange
			"cve-2023-21299":  true, // Exchange
			"cve-2023-21298":  true, // Exchange
			"cve-2023-21297":  true, // Exchange
			"cve-2023-21296":  true, // Exchange
			"cve-2023-21295":  true, // Exchange
			"cve-2023-21294":  true, // Exchange
			"cve-2023-21293":  true, // Exchange
			"cve-2023-21292":  true, // Exchange
			"cve-2023-21291":  true, // Exchange
			"cve-2023-21290":  true, // Exchange
			"cve-2023-21289":  true, // Exchange
			"cve-2023-21288":  true, // Exchange
			"cve-2023-21287":  true, // Exchange
			"cve-2023-21286":  true, // Exchange
			"cve-2023-21285":  true, // Exchange
			"cve-2023-21284":  true, // Exchange
			"cve-2023-21283":  true, // Exchange
			"cve-2023-21282":  true, // Exchange
			"cve-2023-21281":  true, // Exchange
			"cve-2023-21280":  true, // Exchange
			"cve-2023-21279":  true, // Exchange
			"cve-2023-21278":  true, // Exchange
			"cve-2023-21277":  true, // Exchange
			"cve-2023-21276":  true, // Exchange
			"cve-2023-21275":  true, // Exchange
			"cve-2023-21274":  true, // Exchange
			"cve-2023-21273":  true, // Exchange
			"cve-2023-21272":  true, // Exchange
			"cve-2023-21271":  true, // Exchange
			"cve-2023-21270":  true, // Exchange
			"cve-2023-21269":  true, // Exchange
			"cve-2023-21268":  true, // Exchange
			"cve-2023-21267":  true, // Exchange
			"cve-2023-21266":  true, // Exchange
			"cve-2023-21265":  true, // Exchange
			"cve-2023-21264":  true, // Exchange
			"cve-2023-21263":  true, // Exchange
			"cve-2023-21262":  true, // Exchange
			"cve-2023-21261":  true, // Exchange
			"cve-2023-21260":  true, // Exchange
			"cve-2023-21259":  true, // Exchange
			"cve-2023-21258":  true, // Exchange
			"cve-2023-21257":  true, // Exchange
			"cve-2023-21256":  true, // Exchange
			"cve-2023-21255":  true, // Exchange
			"cve-2023-21254":  true, // Exchange
			"cve-2023-21253":  true, // Exchange
			"cve-2023-21252":  true, // Exchange
			"cve-2023-21251":  true, // Exchange
			"cve-2023-21250":  true, // Exchange
			"cve-2023-21249":  true, // Exchange
			"cve-2023-21248":  true, // Exchange
			"cve-2023-21247":  true, // Exchange
			"cve-2023-21246":  true, // Exchange
			"cve-2023-21245":  true, // Exchange
			"cve-2023-21244":  true, // Exchange
			"cve-2023-21243":  true, // Exchange
			"cve-2023-21242":  true, // Exchange
			"cve-2023-21241":  true, // Exchange
			"cve-2023-21240":  true, // Exchange
			"cve-2023-21239":  true, // Exchange
			"cve-2023-21238":  true, // Exchange
			"cve-2023-21237":  true, // Exchange
			"cve-2023-21236":  true, // Exchange
			"cve-2023-21235":  true, // Exchange
			"cve-2023-21234":  true, // Exchange
			"cve-2023-21233":  true, // Exchange
			"cve-2023-21232":  true, // Exchange
			"cve-2023-21231":  true, // Exchange
			"cve-2023-21230":  true, // Exchange
			"cve-2023-21229":  true, // Exchange
			"cve-2023-21228":  true, // Exchange
			"cve-2023-21227":  true, // Exchange
			"cve-2023-21226":  true, // Exchange
			"cve-2023-21225":  true, // Exchange
			"cve-2023-21224":  true, // Exchange
			"cve-2023-21223":  true, // Exchange
			"cve-2023-21222":  true, // Exchange
			"cve-2023-21221":  true, // Exchange
			"cve-2023-21220":  true, // Exchange
			"cve-2023-21219":  true, // Exchange
			"cve-2023-21218":  true, // Exchange
			"cve-2023-21217":  true, // Exchange
			"cve-2023-21216":  true, // Exchange
			"cve-2023-21215":  true, // Exchange
			"cve-2023-21214":  true, // Exchange
			"cve-2023-21213":  true, // Exchange
			"cve-2023-21212":  true, // Exchange
			"cve-2023-21211":  true, // Exchange
			"cve-2023-21210":  true, // Exchange
			"cve-2023-21209":  true, // Exchange
			"cve-2023-21208":  true, // Exchange
			"cve-2023-21207":  true, // Exchange
			"cve-2023-21206":  true, // Exchange
			"cve-2023-21205":  true, // Exchange
			"cve-2023-21204":  true, // Exchange
			"cve-2023-21203":  true, // Exchange
			"cve-2023-21202":  true, // Exchange
			"cve-2023-21201":  true, // Exchange
			"cve-2023-21200":  true, // Exchange
			"cve-2023-21199":  true, // Exchange
			"cve-2023-21198":  true, // Exchange
			"cve-2023-21197":  true, // Exchange
			"cve-2023-21196":  true, // Exchange
			"cve-2023-21195":  true, // Exchange
			"cve-2023-21194":  true, // Exchange
			"cve-2023-21193":  true, // Exchange
			"cve-2023-21192":  true, // Exchange
			"cve-2023-21191":  true, // Exchange
			"cve-2023-21190":  true, // Exchange
			"cve-2023-21189":  true, // Exchange
			"cve-2023-21188":  true, // Exchange
			"cve-2023-21187":  true, // Exchange
			"cve-2023-21186":  true, // Exchange
			"cve-2023-21185":  true, // Exchange
			"cve-2023-21184":  true, // Exchange
			"cve-2023-21183":  true, // Exchange
			"cve-2023-21182":  true, // Exchange
			"cve-2023-21181":  true, // Exchange
			"cve-2023-21180":  true, // Exchange
			"cve-2023-21179":  true, // Exchange
			"cve-2023-21178":  true, // Exchange
			"cve-2023-21177":  true, // Exchange
			"cve-2023-21176":  true, // Exchange
			"cve-2023-21175":  true, // Exchange
			"cve-2023-21174":  true, // Exchange
			"cve-2023-21173":  true, // Exchange
			"cve-2023-21172":  true, // Exchange
			"cve-2023-21171":  true, // Exchange
			"cve-2023-21170":  true, // Exchange
			"cve-2023-21169":  true, // Exchange
			"cve-2023-21168":  true, // Exchange
			"cve-2023-21167":  true, // Exchange
			"cve-2023-21166":  true, // Exchange
			"cve-2023-21165":  true, // Exchange
			"cve-2023-21164":  true, // Exchange
			"cve-2023-21163":  true, // Exchange
			"cve-2023-21162":  true, // Exchange
			"cve-2023-21161":  true, // Exchange
			"cve-2023-21160":  true, // Exchange
			"cve-2023-21159":  true, // Exchange
			"cve-2023-21158":  true, // Exchange
			"cve-2023-21157":  true, // Exchange
			"cve-2023-21156":  true, // Exchange
			"cve-2023-21155":  true, // Exchange
			"cve-2023-21154":  true, // Exchange
			"cve-2023-21153":  true, // Exchange
			"cve-2023-21152":  true, // Exchange
			"cve-2023-21151":  true, // Exchange
			"cve-2023-21150":  true, // Exchange
			"cve-2023-21149":  true, // Exchange
			"cve-2023-21148":  true, // Exchange
			"cve-2023-21147":  true, // Exchange
			"cve-2023-21146":  true, // Exchange
			"cve-2023-21145":  true, // Exchange
			"cve-2023-21144":  true, // Exchange
			"cve-2023-21143":  true, // Exchange
			"cve-2023-21142":  true, // Exchange
			"cve-2023-21141":  true, // Exchange
			"cve-2023-21140":  true, // Exchange
			"cve-2023-21139":  true, // Exchange
			"cve-2023-21138":  true, // Exchange
			"cve-2023-21137":  true, // Exchange
			"cve-2023-21136":  true, // Exchange
			"cve-2023-21135":  true, // Exchange
			"cve-2023-21134":  true, // Exchange
			"cve-2023-21133":  true, // Exchange
			"cve-2023-21132":  true, // Exchange
			"cve-2023-21131":  true, // Exchange
			"cve-2023-21130":  true, // Exchange
			"cve-2023-21129":  true, // Exchange
			"cve-2023-21128":  true, // Exchange
			"cve-2023-21127":  true, // Exchange
			"cve-2023-21126":  true, // Exchange
			"cve-2023-21125":  true, // Exchange
			"cve-2023-21124":  true, // Exchange
			"cve-2023-21123":  true, // Exchange
			"cve-2023-21122":  true, // Exchange
			"cve-2023-21121":  true, // Exchange
			"cve-2023-21120":  true, // Exchange
			"cve-2023-21119":  true, // Exchange
			"cve-2023-21118":  true, // Exchange
			"cve-2023-21117":  true, // Exchange
			"cve-2023-21116":  true, // Exchange
			"cve-2023-21115":  true, // Exchange
			"cve-2023-21114":  true, // Exchange
			"cve-2023-21113":  true, // Exchange
			"cve-2023-21112":  true, // Exchange
			"cve-2023-21111":  true, // Exchange
			"cve-2023-21110":  true, // Exchange
			"cve-2023-21109":  true, // Exchange
			"cve-2023-21108":  true, // Exchange
			"cve-2023-21107":  true, // Exchange
			"cve-2023-21106":  true, // Exchange
			"cve-2023-21105":  true, // Exchange
			"cve-2023-21104":  true, // Exchange
			"cve-2023-21103":  true, // Exchange
			"cve-2023-21102":  true, // Exchange
			"cve-2023-21101":  true, // Exchange
			"cve-2023-21100":  true, // Exchange
			"cve-2023-21099":  true, // Exchange
			"cve-2023-21098":  true, // Exchange
			"cve-2023-21097":  true, // Exchange
			"cve-2023-21096":  true, // Exchange
			"cve-2023-21095":  true, // Exchange
			"cve-2023-21094":  true, // Exchange
			"cve-2023-21093":  true, // Exchange
			"cve-2023-21092":  true, // Exchange
			"cve-2023-21091":  true, // Exchange
			"cve-2023-21090":  true, // Exchange
			"cve-2023-21089":  true, // Exchange
			"cve-2023-21088":  true, // Exchange
			"cve-2023-21087":  true, // Exchange
			"cve-2023-21086":  true, // Exchange
			"cve-2023-21085":  true, // Exchange
			"cve-2023-21084":  true, // Exchange
			"cve-2023-21083":  true, // Exchange
			"cve-2023-21082":  true, // Exchange
			"cve-2023-21081":  true, // Exchange
			"cve-2023-21080":  true, // Exchange
			"cve-2023-21079":  true, // Exchange
			"cve-2023-21078":  true, // Exchange
			"cve-2023-21077":  true, // Exchange
			"cve-2023-21076":  true, // Exchange
			"cve-2023-21075":  true, // Exchange
			"cve-2023-21074":  true, // Exchange
			"cve-2023-21073":  true, // Exchange
			"cve-2023-21072":  true, // Exchange
			"cve-2023-21071":  true, // Exchange
			"cve-2023-21070":  true, // Exchange
			"cve-2023-21069":  true, // Exchange
			"cve-2023-21068":  true, // Exchange
			"cve-2023-21067":  true, // Exchange
			"cve-2023-21066":  true, // Exchange
			"cve-2023-21065":  true, // Exchange
			"cve-2023-21064":  true, // Exchange
			"cve-2023-21063":  true, // Exchange
			"cve-2023-21062":  true, // Exchange
			"cve-2023-21061":  true, // Exchange
			"cve-2023-21060":  true, // Exchange
			"cve-2023-21059":  true, // Exchange
			"cve-2023-21058":  true, // Exchange
			"cve-2023-21057":  true, // Exchange
			"cve-2023-21056":  true, // Exchange
			"cve-2023-21055":  true, // Exchange
			"cve-2023-21054":  true, // Exchange
			"cve-2023-21053":  true, // Exchange
			"cve-2023-21052":  true, // Exchange
			"cve-2023-21051":  true, // Exchange
			"cve-2023-21050":  true, // Exchange
			"cve-2023-21049":  true, // Exchange
			"cve-2023-21048":  true, // Exchange
			"cve-2023-21047":  true, // Exchange
			"cve-2023-21046":  true, // Exchange
			"cve-2023-21045":  true, // Exchange
			"cve-2023-21044":  true, // Exchange
			"cve-2023-21043":  true, // Exchange
			"cve-2023-21042":  true, // Exchange
			"cve-2023-21041":  true, // Exchange
			"cve-2023-21040":  true, // Exchange
			"cve-2023-21039":  true, // Exchange
			"cve-2023-21038":  true, // Exchange
			"cve-2023-21037":  true, // Exchange
			"cve-2023-21036":  true, // Exchange
			"cve-2023-21035":  true, // Exchange
			"cve-2023-21034":  true, // Exchange
			"cve-2023-21033":  true, // Exchange
			"cve-2023-21032":  true, // Exchange
			"cve-2023-21031":  true, // Exchange
			"cve-2023-21030":  true, // Exchange
			"cve-2023-21029":  true, // Exchange
			"cve-2023-21028":  true, // Exchange
			"cve-2023-21027":  true, // Exchange
			"cve-2023-21026":  true, // Exchange
			"cve-2023-21025":  true, // Exchange
			"cve-2023-21024":  true, // Exchange
			"cve-2023-21023":  true, // Exchange
			"cve-2023-21022":  true, // Exchange
			"cve-2023-21021":  true, // Exchange
			"cve-2023-21020":  true, // Exchange
			"cve-2023-21019":  true, // Exchange
			"cve-2023-21018":  true, // Exchange
			"cve-2023-21017":  true, // Exchange
			"cve-2023-21016":  true, // Exchange
			"cve-2023-21015":  true, // Exchange
			"cve-2023-21014":  true, // Exchange
			"cve-2023-21013":  true, // Exchange
			"cve-2023-21012":  true, // Exchange
			"cve-2023-21011":  true, // Exchange
			"cve-2023-21010":  true, // Exchange
			"cve-2023-21009":  true, // Exchange
			"cve-2023-21008":  true, // Exchange
			"cve-2023-21007":  true, // Exchange
			"cve-2023-21006":  true, // Exchange
			"cve-2023-21005":  true, // Exchange
			"cve-2023-21004":  true, // Exchange
			"cve-2023-21003":  true, // Exchange
			"cve-2023-21002":  true, // Exchange
			"cve-2023-21001":  true, // Exchange
			"cve-2023-21000":  true, // Exchange
		},
		popularTargets: map[string]bool{
			"google.com":      true,
			"facebook.com":    true,
			"amazon.com":      true,
			"microsoft.com":   true,
			"apple.com":       true,
			"netflix.com":     true,
			"twitter.com":     true,
			"github.com":      true,
			"linkedin.com":    true,
			"instagram.com":   true,
		},
	}
}

// Assess 评估重复风险
func (a *DuplicateRiskAssessor) Assess(candidate *models.BountyCandidate) string {
	// 检查是否是常见的 CVE 模板
	if a.isCommonCVETemplate(candidate.Title) {
		return models.DuplicateRiskHigh
	}

	// 检查是否是流行目标
	if a.isPopularTarget(candidate) {
		return models.DuplicateRiskMedium
	}

	// 检查漏洞类型
	switch strings.ToLower(candidate.VulnType) {
	case "xss", "csrf", "open_redirect", "info_disclosure":
		return models.DuplicateRiskMedium
	case "rce", "sqli", "file_read", "auth_bypass":
		return models.DuplicateRiskLow
	case "secret_leak", "default_password":
		return models.DuplicateRiskLow
	}

	// 检查来源
	if candidate.SourceKind == models.SourceKindManual {
		return models.DuplicateRiskLow
	}

	return models.DuplicateRiskUnknown
}

// isCommonCVETemplate 检查是否是常见的 CVE 模板
func (a *DuplicateRiskAssessor) isCommonCVETemplate(title string) bool {
	titleLower := strings.ToLower(title)
	for cve := range a.commonCVETemplates {
		if strings.Contains(titleLower, cve) {
			return true
		}
	}
	return false
}

// isPopularTarget 检查是否是流行目标
func (a *DuplicateRiskAssessor) isPopularTarget(candidate *models.BountyCandidate) bool {
	// 这里可以根据项目信息判断
	// 简化处理：检查标题中是否包含流行域名
	titleLower := strings.ToLower(candidate.Title)
	for domain := range a.popularTargets {
		if strings.Contains(titleLower, domain) {
			return true
		}
	}
	return false
}
