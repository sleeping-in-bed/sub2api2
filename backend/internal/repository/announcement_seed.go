package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const initialAnnouncementUpsertSQL = `
INSERT INTO announcements (
	seed_key,
	title,
	content,
	status,
	notify_mode,
	targeting,
	starts_at,
	ends_at,
	created_at,
	updated_at
)
VALUES (
	$1,
	$2,
	$3,
	'active',
	$4,
	'{}'::jsonb,
	$5,
	NULL,
	$5,
	$5
)
ON CONFLICT (seed_key) WHERE seed_key IS NOT NULL
DO UPDATE SET
	title = EXCLUDED.title,
	content = EXCLUDED.content,
	status = EXCLUDED.status,
	notify_mode = EXCLUDED.notify_mode,
	targeting = EXCLUDED.targeting,
	starts_at = EXCLUDED.starts_at,
	ends_at = EXCLUDED.ends_at,
	updated_at = EXCLUDED.updated_at;
`

type initialAnnouncementSeed struct {
	SeedKey    string
	Title      string
	Content    string
	NotifyMode string
	Published  time.Time
}

var initialAnnouncementSeeds = []initialAnnouncementSeed{
	{
		SeedKey:    "codex_5_4_xhigh_supported_2026_03_06",
		Title:      "Codex 5.4 xhigh 系列支持说明",
		NotifyMode: "silent",
		Published:  mustParseAnnouncementSeedTime("2026-03-06T12:34:48+08:00"),
		Content: `平台已支持 Codex 5.4 xhigh 系列模型。由于上游客户端能力仍在分批推送，部分 Codex CLI 或编辑器插件暂时无法在界面中直接选择该系列模型。

如客户端尚未出现可选项，可以在本地 Codex 配置文件中手动指定模型名称。Windows 环境通常位于 C 盘用户目录下的 .codex/config 配置文件；修改后请重启终端或编辑器，使配置重新加载。

需要注意的是，手动指定后客户端一般不会再自动切换模型，默认会按配置中的模型运行。如需恢复原设置，将模型名改回原配置并重启客户端即可。`,
	},
	{
		SeedKey:    "service_stability_and_operation_adjustment_2026_03_13",
		Title:      "近期稳定性说明与运营策略调整",
		NotifyMode: "silent",
		Published:  mustParseAnnouncementSeedTime("2026-03-13T13:02:30+08:00"),
		Content: `近期上游服务出现连续波动，部分模型和渠道受到官方策略调整、容量紧张及风控变化影响，导致请求断流、连接不稳定或可用性下降。我们已经完成多轮渠道排查和接入调整，并会持续观察上游状态，优先保障现有用户的可用性和稳定性。

为了降低突发波动对大家的影响，平台运营策略将转向更谨慎的封闭式维护：新用户注册和公开售卖入口会根据稳定性评估逐步收紧，后续资源会优先用于保障已有用户体验。

服务侧会继续扩容高质量接入节点，并推动高并发用户迁移到支持负载均衡的站点。后续也会评估推出更高规格的模型资源池，以满足更稳定、更高额度的使用需求。

受上游成本变化影响，部分通道的计费倍率会进行调整。我们会尽量保持价格透明，并根据实际稳定性和成本变化继续优化套餐策略。

对于本轮波动造成的不佳体验，平台会为受影响的存量用户安排相应补偿。后续如因平台侧原因导致服务不可用，也会继续按规则处理补偿或退款。`,
	},
	{
		SeedKey:    "service_recovered_and_plan_adjustment_2026_03_30",
		Title:      "服务恢复与套餐调整说明",
		NotifyMode: "silent",
		Published:  mustParseAnnouncementSeedTime("2026-03-30T07:02:36+08:00"),
		Content: `当前服务已基本恢复正常。如果仍遇到 502 或请求不稳定的情况，可以优先将推理强度调整为 high 后重试。

由于上游账号和模型策略变化频繁，平台会继续在后台优化渠道稳定性和调度策略。短期内，月卡和天卡类套餐将暂停新增售卖，后续主要以余额模式提供服务。

我们会持续关注上游风控和可用性变化，在稳定性满足要求后再逐步恢复更多购买和套餐选项。`,
	},
}

func ensureInitialAnnouncements(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("nil sql db")
	}

	for _, seed := range initialAnnouncementSeeds {
		if _, err := db.ExecContext(
			ctx,
			initialAnnouncementUpsertSQL,
			seed.SeedKey,
			seed.Title,
			seed.Content,
			seed.NotifyMode,
			seed.Published,
		); err != nil {
			return fmt.Errorf("seed initial announcement %s: %w", seed.SeedKey, err)
		}
	}

	return nil
}

func mustParseAnnouncementSeedTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
