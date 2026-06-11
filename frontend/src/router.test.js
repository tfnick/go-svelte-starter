import assert from "node:assert/strict";
import { test } from "node:test";

import {
  appRoutes,
  canAccessAppRoute,
  isAppRoute,
  isAuthRoute,
  normalizePath,
  normalizeRouteTarget,
  routeTitle,
  visibleAppRoutes,
} from "./router.js";

test("normalizes app route aliases", () => {
  assert.equal(normalizePath("/index.html"), "/");
  assert.equal(normalizePath("/dashboard"), "/app");
  assert.equal(normalizePath("/orders.html"), "/app/orders");
  assert.equal(normalizePath("/products.html"), "/app/products");
  assert.equal(normalizePath("/users.html"), "/app/users");
  assert.equal(normalizePath("/scheduler.html"), "/app/scheduler");
  assert.equal(normalizePath("/events.html"), "/app/events");
  assert.equal(normalizePath("/experiments.html"), "/app/experiments");
  assert.equal(normalizePath("/dictionary.html"), "/app/dictionary");
  assert.equal(normalizePath("/parameters.html"), "/app/parameters");
  assert.equal(normalizePath("/notifications.html"), "/app/notifications");
  assert.equal(normalizePath("/settings.html"), "/app/settings");
  assert.equal(normalizePath("/variables.html"), "/app/variables");
  assert.equal(normalizePath("/kb-admin.html"), "/app/kb-admin");
  assert.equal(normalizePath("/support-console.html"), "/app/support-console");
  assert.equal(
    normalizeRouteTarget("/orders?tab=mine#latest"),
    "/app/orders?tab=mine#latest",
  );
});

test("exposes logged-in app menu routes from one source", () => {
  assert.deepEqual(
    visibleAppRoutes({ is_admin: true }).map((route) => [
      route.path,
      route.label,
      route.icon,
    ]),
    [
      ["/app", "Dashboard", "dashboard"],
      ["/app/orders", "Order", "orders"],
      ["/app/products", "Product", "products"],
      ["/app/users", "User", "users"],
      ["/app/scheduler", "Scheduler", "scheduler"],
      ["/app/events", "Event", "events"],
      ["/app/experiments", "Experiment", "experiments"],
      ["/app/dictionary", "Dictionary", "dictionary"],
      ["/app/parameters", "Parameter", "parameters"],
      ["/app/notifications", "Notification", "notifications"],
      ["/app/variables", "Variable", "variables"],
      ["/app/settings", "Setting", "settings"],
      ["/app/kb-admin", "Knowledge Base", "book"],
      ["/app/support-console", "Support Console", "support"],
    ],
  );
  assert.equal(
    appRoutes.some((route) => route.path === "/app/checkout" && route.hidden),
    true,
  );
  assert.equal(
    appRoutes.every(
      (route) => typeof route.icon === "string" && route.icon.length > 0,
    ),
    true,
  );
});

test("filters admin-only routes from the menu for regular users", () => {
  assert.equal(
    visibleAppRoutes({ is_admin: false }).some(
      (route) => route.path === "/app/products",
    ),
    false,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: false }).some(
      (route) => route.path === "/app/users",
    ),
    false,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: false }).some(
      (route) => route.path === "/app/scheduler",
    ),
    false,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: false }).some(
      (route) => route.path === "/app/events",
    ),
    false,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: false }).some(
      (route) => route.path === "/app/dictionary",
    ),
    false,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: false }).some(
      (route) => route.path === "/app/parameters",
    ),
    false,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: false }).some(
      (route) => route.path === "/app/notifications",
    ),
    false,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: false }).some(
      (route) => route.path === "/app/variables",
    ),
    false,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: false }).some(
      (route) => route.path === "/app/settings",
    ),
    false,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: true }).some(
      (route) => route.path === "/app/products",
    ),
    true,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: true }).some(
      (route) => route.path === "/app/users",
    ),
    true,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: true }).some(
      (route) => route.path === "/app/scheduler",
    ),
    true,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: true }).some(
      (route) => route.path === "/app/events",
    ),
    true,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: true }).some(
      (route) => route.path === "/app/dictionary",
    ),
    true,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: true }).some(
      (route) => route.path === "/app/parameters",
    ),
    true,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: true }).some(
      (route) => route.path === "/app/notifications",
    ),
    true,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: true }).some(
      (route) => route.path === "/app/variables",
    ),
    true,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: true }).some(
      (route) => route.path === "/app/settings",
    ),
    true,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: 1 }).some(
      (route) => route.path === "/app/notifications",
    ),
    true,
  );
  assert.equal(
    visibleAppRoutes({ is_admin: "1" }).some(
      (route) => route.path === "/app/notifications",
    ),
    true,
  );
});

test("classifies auth and app routes", () => {
  assert.equal(isAuthRoute("/login"), true);
  assert.equal(isAuthRoute("/login/oauth/callback"), true);
  assert.equal(isAuthRoute("/register"), true);
  assert.equal(isAuthRoute("/orders"), false);

  assert.equal(isAppRoute("/"), false);
  assert.equal(isAppRoute("/app"), true);
  assert.equal(isAppRoute("/app/checkout"), true);
  assert.equal(isAppRoute("/app/orders"), true);
  assert.equal(isAppRoute("/app/products"), true);
  assert.equal(isAppRoute("/app/users"), true);
  assert.equal(isAppRoute("/app/scheduler"), true);
  assert.equal(isAppRoute("/app/events"), true);
  assert.equal(isAppRoute("/app/experiments"), true);
  assert.equal(isAppRoute("/app/dictionary"), true);
  assert.equal(isAppRoute("/app/parameters"), true);
  assert.equal(isAppRoute("/app/notifications"), true);
  assert.equal(isAppRoute("/app/settings"), true);
  assert.equal(isAppRoute("/app/variables"), true);
  assert.equal(isAppRoute("/app/kb-admin"), true);
  assert.equal(isAppRoute("/app/support-console"), true);
  assert.equal(isAppRoute("/login"), false);
  assert.equal(isAppRoute("/login/oauth/callback"), false);
  assert.equal(
    canAccessAppRoute("/app/parameters", { is_admin: false }),
    false,
  );
  assert.equal(canAccessAppRoute("/app/checkout", { is_admin: false }), true);
});

test("returns route titles for logged-in menu pages", () => {
  assert.equal(routeTitle("/app"), "Dashboard");
  assert.equal(routeTitle("/app/checkout"), "Checkout");
  assert.equal(routeTitle("/app/orders"), "Order");
  assert.equal(routeTitle("/app/products"), "Product");
  assert.equal(routeTitle("/app/users"), "User");
  assert.equal(routeTitle("/app/scheduler"), "Scheduler");
  assert.equal(routeTitle("/app/events"), "Event");
  assert.equal(routeTitle("/app/experiments"), "Experiment");
  assert.equal(routeTitle("/app/dictionary"), "Dictionary");
  assert.equal(routeTitle("/app/parameters"), "Parameter");
  assert.equal(routeTitle("/app/notifications"), "Notification");
  assert.equal(routeTitle("/app/settings"), "Setting");
  assert.equal(routeTitle("/app/variables"), "Variable");
  assert.equal(routeTitle("/app/kb-admin"), "Knowledge Base");
  assert.equal(routeTitle("/app/support-console"), "Support Console");
  assert.equal(routeTitle("/login/oauth/callback"), "Login");
});
