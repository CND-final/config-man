const baseUrl = process.env.API_BASE_URL || 'http://localhost:3000/api/v1';

async function request(path, options = {}) {
  const response = await fetch(`${baseUrl}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {})
    }
  });
  const text = await response.text();
  const body = text ? JSON.parse(text) : null;

  if (!response.ok) {
    throw new Error(
      `${options.method || 'GET'} ${path} failed with ${response.status}: ${text}`
    );
  }

  return { response, body };
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

async function main() {
  console.log(`API smoke test target: ${baseUrl}`);

  const health = await request('/health');
  assert(health.body.status === 'ok', 'health endpoint should return ok');
  console.log('✓ health endpoint works');

  const login = await request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({
      email: 'admin@config-man.local',
      password: 'password'
    })
  });
  assert(login.body.token, 'login should return a token');
  assert(login.body.user.role === 'system_admin', 'login user should be admin');
  console.log('✓ login endpoint works');

  const token = login.body.token;
  const authHeaders = { Authorization: `Bearer ${token}` };

  const me = await request('/auth/me', { headers: authHeaders });
  assert(me.body.id === login.body.user.id, 'me should return logged-in user');
  console.log('✓ auth/me endpoint works');

  const projects = await request('/projects', { headers: authHeaders });
  assert(Array.isArray(projects.body), 'projects should return an array');
  assert(projects.body.length > 0, 'seeded projects should exist');
  console.log('✓ projects endpoint works');

  const project = projects.body[0];
  const environment = project.environments.some((env) => env.name === 'prod')
    ? 'prod'
    : project.environments[0].name;
  const configs = await request(
    `/projects/${project.id}/configs?env=${environment}`,
    { headers: authHeaders }
  );
  assert(Array.isArray(configs.body.entries), 'configs should return entries');
  console.log('✓ project configs endpoint works');

  const reviews = await request('/review-requests', { headers: authHeaders });
  assert(Array.isArray(reviews.body), 'review requests should return an array');
  console.log('✓ review requests endpoint works');

  console.log('All API smoke tests passed.');
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
