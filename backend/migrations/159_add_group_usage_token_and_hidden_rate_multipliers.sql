ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS input_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS output_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS cache_creation_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS cache_read_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS hidden_input_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS hidden_output_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS hidden_cache_creation_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS hidden_cache_read_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;

ALTER TABLE groups
    ADD CONSTRAINT groups_input_token_multiplier_positive
        CHECK (input_token_multiplier > 0) NOT VALID,
    ADD CONSTRAINT groups_output_token_multiplier_positive
        CHECK (output_token_multiplier > 0) NOT VALID,
    ADD CONSTRAINT groups_cache_creation_token_multiplier_positive
        CHECK (cache_creation_token_multiplier > 0) NOT VALID,
    ADD CONSTRAINT groups_cache_read_token_multiplier_positive
        CHECK (cache_read_token_multiplier > 0) NOT VALID,
    ADD CONSTRAINT groups_hidden_input_rate_multiplier_positive
        CHECK (hidden_input_rate_multiplier > 0) NOT VALID,
    ADD CONSTRAINT groups_hidden_output_rate_multiplier_positive
        CHECK (hidden_output_rate_multiplier > 0) NOT VALID,
    ADD CONSTRAINT groups_hidden_cache_creation_rate_multiplier_positive
        CHECK (hidden_cache_creation_rate_multiplier > 0) NOT VALID,
    ADD CONSTRAINT groups_hidden_cache_read_rate_multiplier_positive
        CHECK (hidden_cache_read_rate_multiplier > 0) NOT VALID;

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS raw_input_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS raw_output_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS raw_cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS raw_cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS group_input_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS group_output_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS group_cache_creation_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS group_cache_read_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS group_hidden_input_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS group_hidden_output_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS group_hidden_cache_creation_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS group_hidden_cache_read_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;

COMMENT ON COLUMN groups.input_token_multiplier IS '输入 Token 记账倍率';
COMMENT ON COLUMN groups.output_token_multiplier IS '输出 Token 记账倍率';
COMMENT ON COLUMN groups.cache_creation_token_multiplier IS '缓存写入 Token 记账倍率';
COMMENT ON COLUMN groups.cache_read_token_multiplier IS '缓存命中 Token 记账倍率';
COMMENT ON COLUMN groups.hidden_input_rate_multiplier IS '隐藏输入费率倍率，仅影响金额';
COMMENT ON COLUMN groups.hidden_output_rate_multiplier IS '隐藏输出费率倍率，仅影响金额';
COMMENT ON COLUMN groups.hidden_cache_creation_rate_multiplier IS '隐藏缓存写入费率倍率，仅影响金额';
COMMENT ON COLUMN groups.hidden_cache_read_rate_multiplier IS '隐藏缓存命中费率倍率，仅影响金额';

COMMENT ON COLUMN usage_logs.raw_input_tokens IS '上游原始输入 token 数';
COMMENT ON COLUMN usage_logs.raw_output_tokens IS '上游原始输出 token 数';
COMMENT ON COLUMN usage_logs.raw_cache_creation_tokens IS '上游原始缓存写入 token 数';
COMMENT ON COLUMN usage_logs.raw_cache_read_tokens IS '上游原始缓存命中 token 数';
COMMENT ON COLUMN usage_logs.group_input_token_multiplier IS '分组输入 Token 记账倍率快照';
COMMENT ON COLUMN usage_logs.group_output_token_multiplier IS '分组输出 Token 记账倍率快照';
COMMENT ON COLUMN usage_logs.group_cache_creation_token_multiplier IS '分组缓存写入 Token 记账倍率快照';
COMMENT ON COLUMN usage_logs.group_cache_read_token_multiplier IS '分组缓存命中 Token 记账倍率快照';
COMMENT ON COLUMN usage_logs.group_hidden_input_rate_multiplier IS '分组隐藏输入费率倍率快照';
COMMENT ON COLUMN usage_logs.group_hidden_output_rate_multiplier IS '分组隐藏输出费率倍率快照';
COMMENT ON COLUMN usage_logs.group_hidden_cache_creation_rate_multiplier IS '分组隐藏缓存写入费率倍率快照';
COMMENT ON COLUMN usage_logs.group_hidden_cache_read_rate_multiplier IS '分组隐藏缓存命中费率倍率快照';
