const aliases = new Map([
  ['/index.html', '/'],
  ['/dashboard', '/app'],
  ['/login', '/app/login'],
  ['/login.html', '/app/login'],
  ['/login/oauth/callback', '/app/login/oauth/callback'],
  ['/login-oauth-callback.html', '/app/login/oauth/callback'],
  ['/register', '/app/register'],
  ['/register.html', '/app/register'],
  ['/forgot-password', '/app/forgot-password'],
  ['/forgot-password.html', '/app/forgot-password'],
  ['/reset-password', '/app/reset-password'],
  ['/reset-password.html', '/app/reset-password'],
  ['/orders', '/app/orders'],
  ['/orders.html', '/app/orders'],
  ['/products', '/app/products'],
  ['/products.html', '/app/products'],
  ['/users', '/app/users'],
  ['/users.html', '/app/users'],
  ['/scheduler', '/app/scheduler'],
  ['/scheduler.html', '/app/scheduler'],
  ['/events', '/app/events'],
  ['/events.html', '/app/events'],
  ['/experiments', '/app/experiments'],
  ['/experiments.html', '/app/experiments'],
  ['/dictionary', '/app/dictionary'],
  ['/dictionary.html', '/app/dictionary'],
  ['/parameters', '/app/parameters'],
  ['/parameters.html', '/app/parameters'],
  ['/notifications', '/app/notifications'],
  ['/notifications.html', '/app/notifications'],
  ['/settings', '/app/settings'],
  ['/settings.html', '/app/settings'],
  ['/variables', '/app/variables'],
  ['/variables.html', '/app/variables']
]);

export const appHomePath = '/app';

export const appRoutes = Object.freeze([
  {
    path: appHomePath,
    label: 'Dashboard',
    description: 'Welcome'
  },
  {
    path: '/app/orders',
    label: 'Order',
    description: 'Orders and points'
  },
  {
    path: '/app/products',
    label: 'Product',
    description: 'Checkout catalog'
  },
  {
    path: '/app/users',
    label: 'User',
    description: 'Accounts'
  },
  {
    path: '/app/scheduler',
    label: 'Scheduler',
    description: 'Reserved'
  },
  {
    path: '/app/events',
    label: 'Event',
    description: 'Domain deliveries'
  },
  {
    path: '/app/experiments',
    label: 'Experiment',
    description: 'LLM and SSE'
  },
  {
    path: '/app/dictionary',
    label: 'Dictionary',
    description: 'Selectable values'
  },
  {
    path: '/app/parameters',
    label: 'Parameter',
    description: 'Integration settings',
    adminOnly: true
  },
  {
    path: '/app/notifications',
    label: 'Notification',
    description: 'Delivery ledger',
    adminOnly: true
  },
  {
    path: '/app/variables',
    label: 'Variable',
    description: 'Global controls'
  },
  {
    path: '/app/settings',
    label: 'Setting',
    description: 'Site preferences',
    adminOnly: true
  },
  {
    path: '/app/checkout',
    label: 'Checkout',
    description: 'Payment handoff',
    hidden: true
  }
]);

export function normalizePath(pathname = window.location.pathname) {
  const pathOnly = String(pathname || '').split(/[?#]/)[0] || '/';
  const normalized = aliases.get(pathOnly) || pathOnly;
  if (normalized === '') {
    return '/';
  }
  return normalized;
}

export function normalizeRouteTarget(value) {
  if (!value || !value.startsWith('/') || value.startsWith('//')) {
    return '';
  }

  try {
    const parsed = new URL(value, 'http://app.local');
    return `${normalizePath(parsed.pathname)}${parsed.search}${parsed.hash}`;
  } catch {
    return '';
  }
}

export function navigate(path) {
  window.history.pushState({}, '', path);
  window.dispatchEvent(new PopStateEvent('popstate'));
}

export function isAuthRoute(path) {
  switch (normalizePath(path)) {
    case '/app/login':
    case '/app/login/oauth/callback':
    case '/app/register':
    case '/app/forgot-password':
    case '/app/reset-password':
      return true;
    default:
      return false;
  }
}

export function isAppRoute(path) {
  const normalized = normalizePath(path);
  return appRoutes.some((route) => route.path === normalized);
}

export function visibleAppRoutes(user = null) {
  const isAdmin = user?.is_admin === true || user?.is_admin === 1 || user?.is_admin === '1' || user?.is_admin === 'true';
  return appRoutes.filter((route) => !route.hidden && (!route.adminOnly || isAdmin));
}

export function canAccessAppRoute(path, user = null) {
  const normalized = normalizePath(path);
  const route = appRoutes.find((candidate) => candidate.path === normalized);
  if (!route) {
    return false;
  }
  const isAdmin = user?.is_admin === true || user?.is_admin === 1 || user?.is_admin === '1' || user?.is_admin === 'true';
  return !route.adminOnly || isAdmin;
}

export function routeTitle(path) {
  switch (normalizePath(path)) {
    case '/app/login':
      return 'Login';
    case '/app/login/oauth/callback':
      return 'Login';
    case '/app/register':
      return 'Register';
    case '/app/forgot-password':
      return 'Forgot Password';
    case '/app/reset-password':
      return 'Reset Password';
    case '/app/orders':
      return 'Order';
    case '/app/products':
      return 'Product';
    case '/app/users':
      return 'User';
    case '/app/scheduler':
      return 'Scheduler';
    case '/app/events':
      return 'Event';
    case '/app/experiments':
      return 'Experiment';
    case '/app/dictionary':
      return 'Dictionary';
    case '/app/parameters':
      return 'Parameter';
    case '/app/notifications':
      return 'Notification';
    case '/app/settings':
      return 'Setting';
    case '/app/variables':
      return 'Variable';
    case '/app/checkout':
      return 'Checkout';
    case '/app':
      return 'Dashboard';
    default:
      return 'Dashboard';
  }
}
