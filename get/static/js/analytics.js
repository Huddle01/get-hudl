// Lightweight PostHog bootstrap for the installer site.
// It stays inert until runtime config provides a project API key.
(function initHudlAnalytics() {
  async function loadConfig() {
    if (window.HUDL_POSTHOG_CONFIG) return window.HUDL_POSTHOG_CONFIG;

    try {
      const response = await fetch('/api/posthog-config', {
        headers: { Accept: 'application/json' },
      });
      if (!response.ok) return null;

      const config = await response.json();
      window.HUDL_POSTHOG_CONFIG = config;
      return config;
    } catch {
      return null;
    }
  }

  function start(config) {
    if (!config || !config.enabled || !config.apiKey) return;

    !function(t,e){var o,n,p,r;e.__SV||(window.posthog=e,e._i=[],e.init=function(i,s,a){function g(t,e){var o=e.split(".");2==o.length&&(t=t[o[0]],e=o[1]),t[e]=function(){t.push([e].concat(Array.prototype.slice.call(arguments,0)))}}(p=t.createElement("script")).type="text/javascript",p.crossOrigin="anonymous",p.async=!0,p.src=s.api_host.replace(".i.posthog.com","-assets.i.posthog.com")+"/static/array.js",(r=t.getElementsByTagName("script")[0]).parentNode.insertBefore(p,r);var u=e;for(void 0!==a?u=e[a]=[]:a="posthog",u.people=u.people||[],u.toString=function(t){var e="posthog";return"posthog"!==a&&(e+="."+a),t||(e+=" (stub)"),e},u.people.toString=function(){return u.toString(1)+".people (stub)"},o="init capture register register_once register_for_session unregister unregister_for_session getFeatureFlag getFeatureFlagPayload isFeatureEnabled reloadFeatureFlags updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures on onFeatureFlags onSessionId getSurveys getActiveMatchingSurveys renderSurvey canRenderSurvey getNextSurveyStep identify setPersonProperties group resetGroups setPersonPropertiesForFlags resetPersonPropertiesForFlags setGroupPropertiesForFlags resetGroupPropertiesForFlags reset get_distinct_id getGroups get_session_id get_session_replay_url alias set_config startSessionRecording stopSessionRecording sessionRecordingStarted captureException loadToolbar get_property getSessionProperty createPersonProfile opt_in_capturing opt_out_capturing has_opted_in_capturing has_opted_out_capturing clear_opt_in_out_capturing debug".split(" "),n=0;n<o.length;n++)g(u,o[n]);e._i.push([i,s,a])},e.__SV=1)}(document,window.posthog||[]);

    window.posthog.init(config.apiKey, {
      api_host: config.apiHost,
      defaults: '2026-01-30',
      autocapture: false,
      capture_pageview: true,
      capture_pageleave: true,
      person_profiles: 'identified_only',
    });

    function capture(eventName, properties) {
      if (!window.posthog || typeof window.posthog.capture !== 'function') return;
      window.posthog.capture(eventName, properties);
    }

    function hrefFor(el) {
      if (!(el instanceof HTMLAnchorElement)) return undefined;
      return el.href || undefined;
    }

    document.addEventListener('click', (event) => {
      const target = event.target.closest('[data-ph-event]');
      if (!target) return;

      capture(target.getAttribute('data-ph-event'), {
        location: target.getAttribute('data-ph-location') || undefined,
        label: target.getAttribute('data-ph-label') || undefined,
        href: hrefFor(target),
      });
    });

    document.addEventListener('hudl:copied', (event) => {
      const detail = event.detail || {};
      capture('install command copied', {
        context: detail.context || 'unknown',
        client: detail.client || 'unknown',
        command: detail.text || '',
      });
    });

    document.addEventListener('hudl:setup-tab-changed', (event) => {
      const detail = event.detail || {};
      capture('setup client selected', {
        client: detail.client || 'unknown',
      });
    });

    window.hudlAnalytics = Object.freeze({ capture });
  }

  loadConfig().then(start);
})();
