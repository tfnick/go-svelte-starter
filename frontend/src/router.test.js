import assert from 'node:assert/strict';
import { test } from 'node:test';

import {
  appRoutes,
  canAccessAppRoute,
  isAppRoute,
  isAuthRoute,
  normalizePath,
  normalizeRouteTarget,
  routeTitle,
  visibleAppRoutes
} from './router.js';

test('normalizes app route aliases', () => {
  assert.equal(normalizePath('/index.html'), '/');
  assert.equal(normalizePath('/dashboard'), '/app');
  assert.equal(normalizePath('/orders.html'), '/app/orders');
  assert.equal(normalizePath('/products.html'), '/app/products');
  assert.equal(normalizePath('/users.html'), '/app/users');
  assert.equal(normalizePath('/scheduler.html'), '/app/scheduler');
  assert.equal(normalizePath('/events.html'), '/app/events');
  assert.equal(normalizePath('/experiments.html'), '/app/experiments');
  assert.equal(normalizePath('/dictionary.html'), '/app/dictionary');
  assert.equal(normalizePath('/parameters.html'), '/app/parameters');
  assert.equal(normalizePath('/notifications.html'), '/app/notifications');
  assert.equal(normalizePath('/settings.html'), '/app/settings');
  assert.equal(normalizePath('/variables.html'), '/app/variables');
  assert.equal(normalizeRouteTarget('/orders?tab=mine#latest'), '/app/orders?tab=mine#latest');
});

test('exposes logged-in app menu routes from one source', () => {
  assert.deepEqual(
    visibleAppRoutes({ is_admin: true }).map((route) => [route.path, route.label]),
    [
      ['/app', 'Dashboard'],
      ['/app/orders', 'Order'],
      ['/app/products', 'Product'],
      ['/app/users', 'User'],
      ['/app/scheduler', 'Scheduler'],
      ['/app/events', 'Event'],
      ['/app/experiments', 'Experiment'],
      ['/app/dictionary', 'Dictionary'],
      ['/app/parameters', 'Parameter'],
      ['/app/notifications', 'Notification'],
      ['/app/variables', 'Variable'],
      ['/app/settings', 'Setting']
    ]
  );
  assert.equal(appRoutes.some((route) => route.path === '/app/checkout' && route.hidden), true);
});

test('filters admin-only routes from the menu for regular users', () => {
  assert.equal(visibleAppRoutes({ is_admin: false }).some((route) => route.path === '/app/parameters'), false);
  assert.equal(visibleAppRoutes({ is_admin: false }).some((route) => route.path === '/app/notifications'), false);
  assert.equal(visibleAppRoutes({ is_admin: false }).some((route) => route.path === '/app/settings'), false);
  assert.equal(visibleAppRoutes({ is_admin: false }).some((route) => route.path === '/app/checkout'), false);
  assert.equal(visibleAppRoutes({ is_admin: true }).some((route) => route.path === '/app/parameters'), true);
  assert.equal(visibleAppRoutes({ is_admin: true }).some((route) => route.path === '/app/notifications'), true);
  assert.equal(visibleAppRoutes({ is_admin: true }).some((route) => route.path === '/app/settings'), true);
  assert.equal(visibleAppRoutes({ is_admin: 1 }).some((route) => route.path === '/app/notifications'), true);
  assert.equal(visibleAppRoutes({ is_admin: '1' }).some((route) => route.path === '/app/notifications'), true);
});

test('classifies auth and app routes', () => {
  assert.equal(isAuthRoute('/login'), true);
  assert.equal(isAuthRoute('/login/oauth/callback'), true);
  assert.equal(isAuthRoute('/register'), true);
  assert.equal(isAuthRoute('/orders'), false);

  assert.equal(isAppRoute('/'), false);
  assert.equal(isAppRoute('/app'), true);
  assert.equal(isAppRoute('/app/checkout'), true);
  assert.equal(isAppRoute('/orders'), true);
  assert.equal(isAppRoute('/app/products'), true);
  assert.equal(isAppRoute('/app/users'), true);
  assert.equal(isAppRoute('/app/scheduler'), true);
  assert.equal(isAppRoute('/app/events'), true);
  assert.equal(isAppRoute('/app/experiments'), true);
  assert.equal(isAppRoute('/app/dictionary'), true);
  assert.equal(isAppRoute('/app/parameters'), true);
  assert.equal(isAppRoute('/app/notifications'), true);
  assert.equal(isAppRoute('/app/settings'), true);
  assert.equal(isAppRoute('/app/variables'), true);
  assert.equal(isAppRoute('/login'), false);
  assert.equal(isAppRoute('/login/oauth/callback'), false);
  assert.equal(canAccessAppRoute('/app/parameters', { is_admin: false }), false);
  assert.equal(canAccessAppRoute('/app/checkout', { is_admin: false }), true);
});

test('returns route titles for logged-in menu pages', () => {
  assert.equal(routeTitle('/app'), 'Dashboard');
  assert.equal(routeTitle('/app/checkout'), 'Checkout');
  assert.equal(routeTitle('/orders'), 'Order');
  assert.equal(routeTitle('/app/products'), 'Product');
  assert.equal(routeTitle('/app/users'), 'User');
  assert.equal(routeTitle('/app/scheduler'), 'Scheduler');
  assert.equal(routeTitle('/app/events'), 'Event');
  assert.equal(routeTitle('/app/experiments'), 'Experiment');
  assert.equal(routeTitle('/app/dictionary'), 'Dictionary');
  assert.equal(routeTitle('/app/parameters'), 'Parameter');
  assert.equal(routeTitle('/app/notifications'), 'Notification');
  assert.equal(routeTitle('/app/settings'), 'Setting');
  assert.equal(routeTitle('/app/variables'), 'Variable');
  assert.equal(routeTitle('/login/oauth/callback'), 'Login');
});
