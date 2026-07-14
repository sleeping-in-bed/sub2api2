INSERT INTO settings (key, value)
VALUES
    ('default_concurrency', '20'),
    ('auth_source_default_email_concurrency', '20'),
    ('auth_source_default_linuxdo_concurrency', '20'),
    ('auth_source_default_oidc_concurrency', '20'),
    ('auth_source_default_wechat_concurrency', '20'),
    ('auth_source_default_github_concurrency', '20'),
    ('auth_source_default_google_concurrency', '20'),
    ('auth_source_default_dingtalk_concurrency', '20')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value;
