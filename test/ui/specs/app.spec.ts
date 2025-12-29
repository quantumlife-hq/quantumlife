import { test, expect } from '@playwright/test';

test.describe('QuantumLife App', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/app');
    // Wait for React to render
    await page.waitForSelector('[class*="gradient-bg"]', { timeout: 10000 });
  });

  test.describe('Navigation', () => {
    test('should display sidebar with all navigation items', async ({ page }) => {
      const navItems = [
        'Dashboard',
        'Inbox',
        'Hats',
        'Recommendations',
        'Learning',
        'Agent Chat',
        'Trust Capital',
        'Audit Trail',
        'Spaces',
        'Settings',
      ];

      for (const item of navItems) {
        await expect(page.getByRole('button', { name: item })).toBeVisible();
      }
    });

    test('should show QuantumLife branding in sidebar', async ({ page }) => {
      await expect(page.getByText('QuantumLife')).toBeVisible();
      await expect(page.getByText('Your Digital Twin')).toBeVisible();
    });

    test('should navigate between views on click', async ({ page }) => {
      // Click Inbox
      await page.getByRole('button', { name: 'Inbox' }).click();
      await expect(page.getByRole('heading', { name: 'Inbox' })).toBeVisible();

      // Click Hats
      await page.getByRole('button', { name: 'Hats' }).click();
      await expect(page.getByRole('heading', { name: 'Your Hats' })).toBeVisible();

      // Click back to Dashboard
      await page.getByRole('button', { name: 'Dashboard' }).click();
      await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
    });
  });

  test.describe('Dashboard View', () => {
    test('should display dashboard stats cards', async ({ page }) => {
      await expect(page.getByText('Total Items')).toBeVisible();
      await expect(page.getByText('Memories')).toBeVisible();
      await expect(page.getByText('Active Hats')).toBeVisible();
      await expect(page.getByText('Agent Status')).toBeVisible();
    });

    test('should show 12 active hats count', async ({ page }) => {
      const hatsCard = page.locator('text=Active Hats').locator('..');
      await expect(hatsCard.getByText('12')).toBeVisible();
    });

    test('should display Your Hats section', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Your Hats' })).toBeVisible();
    });

    test('should display Recent Activity section', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Recent Activity' })).toBeVisible();
    });

    test('should have Refresh button', async ({ page }) => {
      await expect(page.getByRole('button', { name: /Refresh/i })).toBeVisible();
    });
  });

  test.describe('Inbox View', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByRole('button', { name: 'Inbox' }).click();
    });

    test('should display inbox with filters', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Inbox', exact: true })).toBeVisible();
      await expect(page.locator('select').first()).toBeVisible();
    });

    test('should show item count', async ({ page }) => {
      await expect(page.getByText(/\d+ items?/)).toBeVisible();
    });

    test('should show empty state when no items', async ({ page }) => {
      await expect(page.getByText('No items match your filters')).toBeVisible();
    });
  });

  test.describe('Hats View', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByRole('button', { name: 'Hats' }).click();
    });

    test('should display all 12 hats', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Your Hats' })).toBeVisible();

      const hatNames = [
        'Parent', 'Professional', 'Partner', 'Health', 'Finance', 'Learner',
        'Social', 'Home', 'Citizen', 'Creative', 'Spiritual', 'Personal'
      ];

      for (const hat of hatNames) {
        await expect(page.getByText(hat, { exact: false }).first()).toBeVisible();
      }
    });

    test('should show hat descriptions', async ({ page }) => {
      await expect(page.getByText('Children, parenting, school, activities')).toBeVisible();
      await expect(page.getByText('Work, career, colleagues, projects')).toBeVisible();
    });

    test('should display Active and System badges', async ({ page }) => {
      const activeBadges = page.getByText('Active');
      await expect(activeBadges.first()).toBeVisible();

      const systemBadges = page.getByText('System');
      await expect(systemBadges.first()).toBeVisible();
    });
  });

  test.describe('Recommendations View', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByRole('button', { name: 'Recommendations' }).click();
    });

    test('should display recommendations sections', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Recommendations', exact: true })).toBeVisible();
      await expect(page.getByRole('heading', { name: /Active Recommendations/i })).toBeVisible();
      await expect(page.getByRole('heading', { name: /Nudges/i })).toBeVisible();
    });

    test('should show empty state messages', async ({ page }) => {
      await expect(page.getByText('No active recommendations')).toBeVisible();
      await expect(page.getByText('No pending nudges')).toBeVisible();
    });
  });

  test.describe('Learning View', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByRole('button', { name: 'Learning' }).click();
    });

    test('should display behavioral learning page', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Behavioral Learning' })).toBeVisible();
      await expect(page.getByText('How your digital twin understands you')).toBeVisible();
    });

    test('should show understanding score section', async ({ page }) => {
      // Wait for data to load
      await page.waitForSelector('text=Understanding Score', { timeout: 10000 });
      await expect(page.getByText('Understanding Score')).toBeVisible();
      await expect(page.getByText('Signals Captured')).toBeVisible();
    });

    test('should show detected patterns section', async ({ page }) => {
      await page.waitForSelector('text=Detected Patterns', { timeout: 10000 });
      await expect(page.getByText('Detected Patterns')).toBeVisible();
    });

    test('should show sender insights section', async ({ page }) => {
      await page.waitForSelector('text=Sender Insights', { timeout: 10000 });
      await expect(page.getByText('Sender Insights')).toBeVisible();
    });
  });

  test.describe('Agent Chat View', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByRole('button', { name: 'Agent Chat' }).click();
    });

    test('should display chat interface', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Agent Chat' })).toBeVisible();
      await expect(page.getByText('Talk to your digital twin')).toBeVisible();
    });

    test('should show empty chat state', async ({ page }) => {
      await expect(page.getByText('Start a conversation with your AI agent')).toBeVisible();
    });

    test('should have message input', async ({ page }) => {
      await expect(page.getByPlaceholder('Type a message...')).toBeVisible();
    });

    test('should have send button', async ({ page }) => {
      const sendButton = page.locator('button').filter({ has: page.locator('svg') }).last();
      await expect(sendButton).toBeVisible();
    });
  });

  test.describe('Trust Capital View', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByRole('button', { name: 'Trust Capital' }).click();
    });

    test('should display trust capital page', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Trust Capital' })).toBeVisible();
      await expect(page.getByText('Monitor and manage agent trust across domains')).toBeVisible();
    });

    test('should show overall trust section', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Overall Trust' })).toBeVisible();
      await expect(page.getByText('Trust Score', { exact: true })).toBeVisible();
      await expect(page.getByText('Active Domains')).toBeVisible();
    });

    test('should display trust score value', async ({ page }) => {
      // Default trust score is 50.0
      await expect(page.getByText('50.0')).toBeVisible();
    });

    test('should show domain trust scores section', async ({ page }) => {
      await expect(page.getByText('Domain Trust Scores')).toBeVisible();
    });

    test('should display trust states legend', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Trust States' })).toBeVisible();
      // Check for state badges
      await expect(page.locator('span').filter({ hasText: 'Probation' }).first()).toBeVisible();
      await expect(page.locator('span').filter({ hasText: /^Learning$/ }).first()).toBeVisible();
      await expect(page.locator('span').filter({ hasText: 'Trusted' }).first()).toBeVisible();
      await expect(page.locator('span').filter({ hasText: 'Verified' }).first()).toBeVisible();
      await expect(page.locator('span').filter({ hasText: 'Restricted' }).first()).toBeVisible();
    });

    test('should show state descriptions', async ({ page }) => {
      await expect(page.getByText('New agent, suggestions only')).toBeVisible();
      await expect(page.getByText('Building trust, supervised mode')).toBeVisible();
      await expect(page.getByText('Autonomous with undo window')).toBeVisible();
      await expect(page.getByText('Full autonomy')).toBeVisible();
      await expect(page.getByText('Trust lost, recovery required')).toBeVisible();
    });

    test('should show current trust state indicator', async ({ page }) => {
      // Default state is Learning
      await expect(page.getByText('Learning - Supervised operation, building trust')).toBeVisible();
    });
  });

  test.describe('Audit Trail View', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByRole('button', { name: 'Audit Trail' }).click();
    });

    test('should display audit trail page', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Audit Trail' })).toBeVisible();
      await expect(page.getByText('Cryptographic hash-chained ledger of all agent actions')).toBeVisible();
    });

    test('should show chain integrity status', async ({ page }) => {
      // Either "Chain Integrity Verified" or "Chain Integrity Failed" depending on entries
      const integrityStatus = page.getByText(/Chain Integrity (Verified|Failed)/);
      await expect(integrityStatus).toBeVisible();
    });

    test('should display entry statistics', async ({ page }) => {
      await expect(page.getByText('Total Entries')).toBeVisible();
      await expect(page.getByText('First Entry')).toBeVisible();
      await expect(page.getByText('Last Entry')).toBeVisible();
      await expect(page.getByText('Chain Valid')).toBeVisible();
    });

    test('should show chain valid status', async ({ page }) => {
      await expect(page.getByText('Yes')).toBeVisible();
    });

    test('should show recent entries section', async ({ page }) => {
      await expect(page.getByText('Recent Entries')).toBeVisible();
    });

    test('should show empty state when no entries', async ({ page }) => {
      await expect(page.getByText('No ledger entries yet')).toBeVisible();
      await expect(page.getByText('Entries will appear as the agent performs actions')).toBeVisible();
    });
  });

  test.describe('Spaces View', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByRole('button', { name: 'Spaces' }).click();
    });

    test('should display connected spaces page', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Connected Spaces' })).toBeVisible();
      await expect(page.getByText('Integrations and data sources')).toBeVisible();
    });

    test('should show empty state with CLI hint', async ({ page }) => {
      await expect(page.getByText('No spaces connected')).toBeVisible();
      await expect(page.getByText('Connect your first space using the CLI')).toBeVisible();
      await expect(page.getByText('ql spaces add gmail')).toBeVisible();
    });
  });

  test.describe('Settings View', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByRole('button', { name: 'Settings' }).click();
    });

    test('should display settings page', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Settings', exact: true })).toBeVisible();
      await expect(page.getByText('Configure your digital twin')).toBeVisible();
    });

    test('should show settings navigation tabs', async ({ page }) => {
      await expect(page.getByRole('button', { name: /Profile/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /Agent Behavior/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /Hat Settings/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /Notifications/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /Privacy/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /About/i })).toBeVisible();
    });

    test('should show profile settings by default', async ({ page }) => {
      await expect(page.getByRole('heading', { name: 'Profile Settings' })).toBeVisible();
      await expect(page.getByText('Display Name')).toBeVisible();
      await expect(page.getByText('Timezone')).toBeVisible();
    });

    test('should display user name in profile', async ({ page }) => {
      // User name is "Satish" from screenshots
      const nameInput = page.locator('input').first();
      await expect(nameInput).toBeVisible();
    });
  });

  test.describe('API Integration', () => {
    test('should fetch stats from API', async ({ page }) => {
      const response = await page.request.get('/api/v1/stats');
      expect(response.ok()).toBeTruthy();

      const data = await response.json();
      expect(data).toHaveProperty('total_items');
      expect(data).toHaveProperty('total_memories');
      expect(data).toHaveProperty('trust');
    });

    test('should fetch hats from API', async ({ page }) => {
      const response = await page.request.get('/api/v1/hats');
      expect(response.ok()).toBeTruthy();

      const data = await response.json();
      expect(Array.isArray(data)).toBeTruthy();
      expect(data.length).toBe(12);
    });

    test('should fetch trust data from API', async ({ page }) => {
      const response = await page.request.get('/api/v1/trust');
      expect(response.ok()).toBeTruthy();
    });

    test('should fetch items from API', async ({ page }) => {
      const response = await page.request.get('/api/v1/items');
      expect(response.ok()).toBeTruthy();
    });
  });

  test.describe('Responsive Design', () => {
    test('should render correctly on mobile viewport', async ({ page }) => {
      await page.setViewportSize({ width: 375, height: 667 });
      await page.reload();

      // App should still load
      await expect(page.getByText('QuantumLife')).toBeVisible();
    });

    test('should render correctly on tablet viewport', async ({ page }) => {
      await page.setViewportSize({ width: 768, height: 1024 });
      await page.reload();

      await expect(page.getByText('QuantumLife')).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
    });
  });
});
