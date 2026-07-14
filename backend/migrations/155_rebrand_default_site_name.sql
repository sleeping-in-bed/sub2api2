UPDATE settings
SET value = 'MagaAI',
    updated_at = NOW()
WHERE key = 'site_name'
  AND (value = '' OR value = 'Sub2API');
