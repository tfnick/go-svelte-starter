const aliases = new Map([
  ['/index.html', '/'],
  ['/dashboard', '/'],
  ['/login.html', '/login'],
  ['/register.html', '/register'],
  ['/forgot-password.html', '/forgot-password'],
  ['/reset-password.html', '/reset-password'],
  ['/orders.html', '/orders'],
  ['/products.html', '/products'],
  ['/users.html', '/users'],
  ['/scheduler.html', '/scheduler'],
  ['/events.html', '/events'],
  ['/experiments.html', '/experiments'],
  ['/dictionary.html', '/dictionary'],
  ['/parameters.html', '/parameters'],
  ['/notifications.html', '/notifications'],
  ['/settings.html', '/settings'],
  ['/variables.html', '/variables']
]);

export const appRoutes = Object.freeze([
  {
    path: '/',
    label: 'Dashboard',
    description: 'Welcome'
  },
  {
    path: '/orders',
    label: 'Order',
    description: 'Orders and points'
  },
  {
    path: '/products',
    label: 'Product',
    description: 'Checkout catalog'
  },
  {
    path: '/users',
    label: 'User',
    description: 'Accounts'
  },
  {
    path: '/scheduler',
    label: 'Scheduler',
    description: 'Reserved'
  },
  {
    path: '/events',
    label: 'Event',
    description: 'Domain deliveries'
  },
  {
    path: '/experiments',
    label: 'Experiment',
    description: 'LLM and SSE'
  },
  {
    path: '/dictionary',
    label: 'Dictionary',
    description: 'Selectable values'
  },
  {
    path: '/parameters',
    label: 'Parameter',
    description: 'Integration settings',
    adminOnly: true
  },
  {
    path: '/notifications',
    label: 'Notification',
    description: 'Delivery ledger',
    adminOnly: true
  },
  {
    path: '/variables',
    label: 'Variable',
    description: 'Global controls'
  },
  {
    path: '/settings',
    label: 'Setting',
    description: 'Site preferences',
    adminOnly: true
  }
]);

export function normalizePath(pathname = window.location.pathname) {
  const normalized = aliases.get(pathname) || pathname;
  if (normalized === '') {
    return '/';
  }
  return normalized;
}

export function navigate(path) {
  window.history.pushState({}, '', path);
  window.dispatchEvent(new PopStateEvent('popstate'));
}

export function isAuthRoute(path) {
  switch (normalizePath(path)) {
    case '/login':
    case '/register':
    case '/forgot-password':
    case '/reset-password':
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
  return appRoutes.filter((route) => !route.adminOnly || isAdmin);
}

export function routeTitle(path) {
  switch (normalizePath(path)) {
    case '/login':
      return 'Login';
    case '/register':
      return 'Register';
    case '/forgot-password':
      return 'Forgot Password';
    case '/reset-password':
      return 'Reset Password';
    case '/orders':
      return 'Order';
    case '/products':
      return 'Product';
    case '/users':
      return 'User';
    case '/scheduler':
      return 'Scheduler';
    case '/events':
      return 'Event';
    case '/experiments':
      return 'Experiment';
    case '/dictionary':
      return 'Dictionary';
    case '/parameters':
      return 'Parameter';
    case '/notifications':
      return 'Notification';
    case '/settings':
      return 'Setting';
    case '/variables':
      return 'Variable';
    case '/':
      return 'Dashboard';
    default:
      return 'Dashboard';
  }
}
