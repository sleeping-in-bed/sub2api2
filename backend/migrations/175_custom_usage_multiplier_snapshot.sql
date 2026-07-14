ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS account_input_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS account_output_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS account_cache_creation_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS account_cache_read_token_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    ADD COLUMN IF NOT EXISTS multiplier_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE OR REPLACE FUNCTION sync_usage_log_multiplier_snapshot()
RETURNS trigger AS $$
BEGIN
    NEW.raw_input_tokens := COALESCE((NEW.multiplier_snapshot->>'raw_input_tokens')::integer, NEW.raw_input_tokens);
    NEW.raw_output_tokens := COALESCE((NEW.multiplier_snapshot->>'raw_output_tokens')::integer, NEW.raw_output_tokens);
    NEW.raw_cache_creation_tokens := COALESCE((NEW.multiplier_snapshot->>'raw_cache_creation_tokens')::integer, NEW.raw_cache_creation_tokens);
    NEW.raw_cache_read_tokens := COALESCE((NEW.multiplier_snapshot->>'raw_cache_read_tokens')::integer, NEW.raw_cache_read_tokens);
    NEW.group_input_token_multiplier := COALESCE((NEW.multiplier_snapshot->>'group_input_token_multiplier')::numeric, NEW.group_input_token_multiplier);
    NEW.group_output_token_multiplier := COALESCE((NEW.multiplier_snapshot->>'group_output_token_multiplier')::numeric, NEW.group_output_token_multiplier);
    NEW.group_cache_creation_token_multiplier := COALESCE((NEW.multiplier_snapshot->>'group_cache_creation_token_multiplier')::numeric, NEW.group_cache_creation_token_multiplier);
    NEW.group_cache_read_token_multiplier := COALESCE((NEW.multiplier_snapshot->>'group_cache_read_token_multiplier')::numeric, NEW.group_cache_read_token_multiplier);
    NEW.group_hidden_input_rate_multiplier := COALESCE((NEW.multiplier_snapshot->>'group_hidden_input_rate_multiplier')::numeric, NEW.group_hidden_input_rate_multiplier);
    NEW.group_hidden_output_rate_multiplier := COALESCE((NEW.multiplier_snapshot->>'group_hidden_output_rate_multiplier')::numeric, NEW.group_hidden_output_rate_multiplier);
    NEW.group_hidden_cache_creation_rate_multiplier := COALESCE((NEW.multiplier_snapshot->>'group_hidden_cache_creation_rate_multiplier')::numeric, NEW.group_hidden_cache_creation_rate_multiplier);
    NEW.group_hidden_cache_read_rate_multiplier := COALESCE((NEW.multiplier_snapshot->>'group_hidden_cache_read_rate_multiplier')::numeric, NEW.group_hidden_cache_read_rate_multiplier);
    NEW.account_input_token_multiplier := COALESCE((NEW.multiplier_snapshot->>'account_input_token_multiplier')::numeric, NEW.account_input_token_multiplier);
    NEW.account_output_token_multiplier := COALESCE((NEW.multiplier_snapshot->>'account_output_token_multiplier')::numeric, NEW.account_output_token_multiplier);
    NEW.account_cache_creation_token_multiplier := COALESCE((NEW.multiplier_snapshot->>'account_cache_creation_token_multiplier')::numeric, NEW.account_cache_creation_token_multiplier);
    NEW.account_cache_read_token_multiplier := COALESCE((NEW.multiplier_snapshot->>'account_cache_read_token_multiplier')::numeric, NEW.account_cache_read_token_multiplier);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS usage_logs_multiplier_snapshot_sync ON usage_logs;
CREATE TRIGGER usage_logs_multiplier_snapshot_sync
BEFORE INSERT OR UPDATE OF multiplier_snapshot ON usage_logs
FOR EACH ROW EXECUTE FUNCTION sync_usage_log_multiplier_snapshot();
