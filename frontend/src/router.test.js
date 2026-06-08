import assert from 'node:assert/strict';
import { test } from 'node:test';

import { appRoutes, isAppRoute, isAuthRoute, normalizePath, routeTitle, visibleAppRoutes } from './router.js';

test('normalizes app route aliases', () => {
  assert.equal(normalizePath('/index.html'), '/');
  assert.equal(normalizePath('/dashboard'), '/');
  assert.equal(normalizePath('/orders.html'), '/orders');
  assert.equal(normalizePath('/products.html'), '/products');
  assert.equal(normalizePath('/users.html'), '/users');
  assert.equal(normalizePath('/scheduler.html'), '/scheduler');
  assert.equal(normalizePath('/events.html'), '/events');
  assert.equal(normalizePath('/experiments.html'), '/experiments');
  assert.equal(normalizePath('/dictionary.html'), '/dictionary');
  assert.equal(normalizePath('/parameters.html'), '/parameters');
  assert.equal(normalizePath('/notifications.html'), '/notifications');
  assert.equal(normalizePath('/settings.html'), '/settings');
  assert.equal(normalizePath('/variables.html'), '/variables');
});

test('exposes logged-in app menu routes from one source', () => {
  assert.deepEqual(
    appRoutes.map((route) => [route.path, route.label]),
    [
      ['/', 'Dashboard'],
      ['/orders', 'Order'],
      ['/products', 'Product'],
      ['/users', 'User'],
      ['/scheduler', 'Scheduler'],
      ['/events', 'Event'],
      ['/experiments', 'Experiment'],
      ['/dictionary', 'Dictionary'],
      ['/parameters', 'Parameter'],
      ['/notifications', 'Notification'],
      ['/variables', 'Variable'],
      ['/settings', 'Setting']
    ]
  );
});

test('filters admin-only routes from the menu for regular users', () => {
  assert.equal(visibleAppRoutes({ is_admin: false }).some((route) => route.path === '/parameters'), false);
  assert.equal(visibleAppRoutes({ is_admin: false }).some((route) => route.path === '/notifications'), false);
  assert.equal(visibleAppRoutes({ is_admin: false }).some((route) => route.path === '/settings'), false);
  assert.equal(visibleAppRoutes({ is_admin: true }).some((route) => route.path === '/parameters'), true);
  assert.equal(visibleAppRoutes({ is_admin: true }).some((route) => route.path === '/notifications'), true);
  assert.equal(visibleAppRoutes({ is_admin: true }).some((route) => route.path === '/settings'), true);
  assert.equal(visibleAppRoutes({ is_admin: 1 }).some((route) => route.path === '/notifications'), true);
  assert.equal(visibleAppRoutes({ is_admin: '1' }).some((route) => route.path === '/notifications'), true);
});

test('classifies auth and app routes', () => {
  assert.equal(isAuthRoute('/login'), true);
  assert.equal(isAuthRoute('/login/oauth/callback'), true);
  assert.equal(isAuthRoute('/register'), true);
  assert.equal(isAuthRoute('/orders'), false);

  assert.equal(isAppRoute('/'), true);
  assert.equal(isAppRoute('/orders'), true);
  assert.equal(isAppRoute('/products'), true);
  assert.equal(isAppRoute('/users'), true);
  assert.equal(isAppRoute('/scheduler'), true);
  assert.equal(isAppRoute('/events'), true);
  assert.equal(isAppRoute('/experiments'), true);
  assert.equal(isAppRoute('/dictionary'), true);
  assert.equal(isAppRoute('/parameters'), true);
  assert.equal(isAppRoute('/notifications'), true);
  assert.equal(isAppRoute('/settings'), true);
  assert.equal(isAppRoute('/variables'), true);
  assert.equal(isAppRoute('/login'), false);
  assert.equal(isAppRoute('/login/oauth/callback'), false);
});

test('returns route titles for logged-in menu pages', () => {
  assert.equal(routeTitle('/'), 'Dashboard');
  assert.equal(routeTitle('/orders'), 'Order');
  assert.equal(routeTitle('/products'), 'Product');
  assert.equal(routeTitle('/users'), 'User');
  assert.equal(routeTitle('/scheduler'), 'Scheduler');
  assert.equal(routeTitle('/events'), 'Event');
  assert.equal(routeTitle('/experiments'), 'Experiment');
  assert.equal(routeTitle('/dictionary'), 'Dictionary');
  assert.equal(routeTitle('/parameters'), 'Parameter');
  assert.equal(routeTitle('/notifications'), 'Notification');
  assert.equal(routeTitle('/settings'), 'Setting');
  assert.equal(routeTitle('/variables'), 'Variable');
  assert.equal(routeTitle('/login/oauth/callback'), 'Login');
});
