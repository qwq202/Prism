
const SERVICE_NAME = 'prism';

function isAppRuntimeCache(name) {
  return (
    name === SERVICE_NAME ||
    name.startsWith('coai-') ||
    name.startsWith('prism-') ||
    name.startsWith('workbox-')
  );
}

self.addEventListener('install', function (event) {
  event.waitUntil(self.skipWaiting());
});

self.addEventListener('activate', function (event) {
  event.waitUntil(
    caches.keys()
      .then(function (keys) {
        return Promise.all(
          keys
            .filter(isAppRuntimeCache)
            .map(function (key) {
              return caches.delete(key);
            })
        );
      })
      .then(function () {
        return self.clients.claim();
      })
  );

  console.debug('[service] service worker activated');
});
