'use strict';

module.exports = function handler(_req, res) {
  const apiKey = process.env.HUDL_POSTHOG_API_KEY || '';
  const apiHost = process.env.HUDL_POSTHOG_API_HOST || 'https://us.i.posthog.com';

  res.setHeader('Content-Type', 'application/json; charset=utf-8');
  res.setHeader('Cache-Control', 'no-store, max-age=0');
  res.status(200).json({
    enabled: Boolean(apiKey),
    apiKey,
    apiHost,
  });
};
