import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import { clsx } from 'clsx'
import {
  Cog6ToothIcon,
  UserIcon,
  BellIcon,
  ShieldCheckIcon,
  PaintBrushIcon,
  KeyIcon,
  TrashIcon,
  PlusIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/services/api'
import { PageContent, PageSection } from '@/components/layout'
import {
  Card,
  CardHeader,
  CardContent,
  CardFooter,
  Button,
  Input,
  Textarea,
  Badge,
  Modal,
  ModalBody,
  ModalFooter,
  ConfirmModal,
  useToast,
} from '@/components/ui'
import { useAppStore } from '@/stores/app'
import type { Hat, Settings } from '@/types'

type SettingsTab = 'profile' | 'notifications' | 'autonomy' | 'hats' | 'appearance' | 'danger'

const tabs: Array<{ id: SettingsTab; label: string; icon: React.ComponentType<{ className?: string }> }> = [
  { id: 'profile', label: 'Profile', icon: UserIcon },
  { id: 'notifications', label: 'Notifications', icon: BellIcon },
  { id: 'autonomy', label: 'Autonomy', icon: ShieldCheckIcon },
  { id: 'hats', label: 'Hats', icon: KeyIcon },
  { id: 'appearance', label: 'Appearance', icon: PaintBrushIcon },
  { id: 'danger', label: 'Danger Zone', icon: TrashIcon },
]

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('profile')

  return (
    <PageContent maxWidth="5xl">
      <PageSection>
        <Card className="bg-gradient-to-r from-gray-500/20 via-slate-500/20 to-zinc-500/20 border-gray-500/30">
          <div className="flex items-center gap-4">
            <div className="w-12 h-12 rounded-xl bg-gray-500/30 flex items-center justify-center">
              <Cog6ToothIcon className="w-6 h-6 text-gray-400" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">Settings</h2>
              <p className="text-gray-400">
                Manage your account and preferences
              </p>
            </div>
          </div>
        </Card>
      </PageSection>

      <div className="flex gap-6">
        {/* Sidebar */}
        <nav className="w-56 flex-shrink-0">
          <ul className="space-y-1">
            {tabs.map((tab) => (
              <li key={tab.id}>
                <button
                  onClick={() => setActiveTab(tab.id)}
                  className={clsx(
                    'w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-left transition-all',
                    activeTab === tab.id
                      ? 'bg-primary-500/20 text-primary-300 border border-primary-500/30'
                      : 'text-gray-400 hover:text-white hover:bg-surface-light/30 border border-transparent'
                  )}
                >
                  <tab.icon className="w-5 h-5" />
                  <span className="text-sm font-medium">{tab.label}</span>
                </button>
              </li>
            ))}
          </ul>
        </nav>

        {/* Content */}
        <div className="flex-1">
          <motion.div
            key={activeTab}
            initial={{ opacity: 0, x: 10 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.2 }}
          >
            {activeTab === 'profile' && <ProfileSettings />}
            {activeTab === 'notifications' && <NotificationSettings />}
            {activeTab === 'autonomy' && <AutonomySettings />}
            {activeTab === 'hats' && <HatsSettings />}
            {activeTab === 'appearance' && <AppearanceSettings />}
            {activeTab === 'danger' && <DangerZone />}
          </motion.div>
        </div>
      </div>
    </PageContent>
  )
}

// Profile Settings
function ProfileSettings() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { identity, setIdentity } = useAppStore()
  const [formData, setFormData] = useState({
    name: identity?.name || '',
    email: identity?.email || '',
    bio: identity?.bio || '',
  })

  const updateMutation = useMutation({
    mutationFn: (data: typeof formData) => api.identity.update(data),
    onSuccess: (updatedIdentity) => {
      setIdentity(updatedIdentity)
      queryClient.invalidateQueries({ queryKey: ['identity'] })
      toast.success('Profile updated', 'Your changes have been saved')
    },
    onError: () => {
      toast.error('Update failed', 'Please try again')
    },
  })

  return (
    <Card>
      <CardHeader title="Profile" subtitle="Update your personal information" />
      <CardContent>
        <div className="space-y-4">
          <Input
            label="Name"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          />
          <Input
            label="Email"
            type="email"
            value={formData.email}
            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
          />
          <Textarea
            label="Bio"
            value={formData.bio}
            onChange={(e) => setFormData({ ...formData, bio: e.target.value })}
            rows={3}
          />
        </div>
      </CardContent>
      <CardFooter>
        <Button
          onClick={() => updateMutation.mutate(formData)}
          isLoading={updateMutation.isPending}
        >
          Save Changes
        </Button>
      </CardFooter>
    </Card>
  )
}

// Notification Settings
function NotificationSettings() {
  const toast = useToast()
  const { settings, setSettings } = useAppStore()
  const [notifications, setNotifications] = useState({
    enabled: settings?.notifications ?? true,
    suggestions: true,
    actions: true,
    trust: true,
  })

  const handleSave = async () => {
    await api.settings.update({ notifications: notifications.enabled })
    setSettings({ ...settings, notifications: notifications.enabled })
    toast.success('Settings saved', 'Notification preferences updated')
  }

  return (
    <Card>
      <CardHeader title="Notifications" subtitle="Control how you receive updates" />
      <CardContent>
        <div className="space-y-4">
          <ToggleItem
            label="Enable notifications"
            description="Receive push notifications from QuantumLife"
            checked={notifications.enabled}
            onChange={(checked) => setNotifications({ ...notifications, enabled: checked })}
          />
          <ToggleItem
            label="AI Suggestions"
            description="Get notified when AI has suggestions for you"
            checked={notifications.suggestions}
            onChange={(checked) => setNotifications({ ...notifications, suggestions: checked })}
            disabled={!notifications.enabled}
          />
          <ToggleItem
            label="Action Completions"
            description="Notify when automated actions are completed"
            checked={notifications.actions}
            onChange={(checked) => setNotifications({ ...notifications, actions: checked })}
            disabled={!notifications.enabled}
          />
          <ToggleItem
            label="Trust Updates"
            description="Get updates about your trust level changes"
            checked={notifications.trust}
            onChange={(checked) => setNotifications({ ...notifications, trust: checked })}
            disabled={!notifications.enabled}
          />
        </div>
      </CardContent>
      <CardFooter>
        <Button onClick={handleSave}>Save Changes</Button>
      </CardFooter>
    </Card>
  )
}

// Autonomy Settings
function AutonomySettings() {
  const toast = useToast()
  const { settings, setSettings } = useAppStore()
  const [autonomyMode, setAutonomyMode] = useState(settings?.autonomyMode || 'suggest')

  const modes = [
    {
      id: 'ask',
      name: 'Ask First',
      description: 'AI will always ask before taking any action',
      icon: '🛡️',
    },
    {
      id: 'suggest',
      name: 'Suggest & Approve',
      description: 'AI suggests actions, you approve or reject',
      icon: '💡',
    },
    {
      id: 'auto',
      name: 'Auto-pilot',
      description: 'AI acts autonomously within your trust limits',
      icon: '🚀',
    },
  ]

  const handleSave = async () => {
    await api.settings.update({ autonomyMode })
    setSettings({ ...settings, autonomyMode })
    toast.success('Settings saved', 'Autonomy mode updated')
  }

  return (
    <Card>
      <CardHeader title="Autonomy Mode" subtitle="Control how much autonomy to give your AI assistant" />
      <CardContent>
        <div className="space-y-3">
          {modes.map((mode) => (
            <button
              key={mode.id}
              onClick={() => setAutonomyMode(mode.id as typeof autonomyMode)}
              className={clsx(
                'w-full flex items-start gap-4 p-4 rounded-xl border transition-all text-left',
                autonomyMode === mode.id
                  ? 'bg-primary-500/20 border-primary-500/50'
                  : 'bg-surface-light/20 border-surface-border hover:border-gray-600'
              )}
            >
              <span className="text-2xl">{mode.icon}</span>
              <div className="flex-1">
                <span className="font-medium text-white">{mode.name}</span>
                <p className="text-sm text-gray-400 mt-1">{mode.description}</p>
              </div>
            </button>
          ))}
        </div>
      </CardContent>
      <CardFooter>
        <Button onClick={handleSave}>Save Changes</Button>
      </CardFooter>
    </Card>
  )
}

// Hats Settings
function HatsSettings() {
  const toast = useToast()
  const queryClient = useQueryClient()
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [newHat, setNewHat] = useState({ name: '', icon: '👤', description: '' })

  const { data: hats, isLoading } = useQuery({
    queryKey: ['hats'],
    queryFn: api.hats.list,
  })

  const createMutation = useMutation({
    mutationFn: (hat: typeof newHat) => api.hats.create(hat),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['hats'] })
      toast.success('Hat created', 'Your new hat has been added')
      setShowCreateModal(false)
      setNewHat({ name: '', icon: '👤', description: '' })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (hatId: string) => api.hats.delete(hatId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['hats'] })
      toast.success('Hat deleted', 'The hat has been removed')
    },
  })

  return (
    <>
      <Card>
        <CardHeader
          title="Hats"
          subtitle="Manage your context identities"
          action={
            <Button size="sm" onClick={() => setShowCreateModal(true)}>
              <PlusIcon className="w-4 h-4 mr-2" />
              Add Hat
            </Button>
          }
        />
        <CardContent>
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-16 bg-surface-lighter rounded-lg animate-pulse" />
              ))}
            </div>
          ) : !hats?.length ? (
            <div className="text-center py-8 text-gray-500">
              No hats configured. Create one to get started.
            </div>
          ) : (
            <div className="space-y-3">
              {hats.map((hat) => (
                <div
                  key={hat.id}
                  className="flex items-center justify-between p-4 rounded-lg bg-surface-light/20 border border-surface-border"
                >
                  <div className="flex items-center gap-3">
                    <span className="text-2xl">{hat.icon}</span>
                    <div>
                      <div className="font-medium text-white">{hat.name}</div>
                      <div className="text-sm text-gray-500">{hat.description}</div>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => deleteMutation.mutate(hat.id)}
                  >
                    <TrashIcon className="w-4 h-4 text-red-400" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Modal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title="Create Hat"
        size="sm"
      >
        <ModalBody>
          <div className="space-y-4">
            <Input
              label="Hat Name"
              value={newHat.name}
              onChange={(e) => setNewHat({ ...newHat, name: e.target.value })}
              placeholder="e.g., Work, Personal, Side Project"
            />
            <Input
              label="Icon"
              value={newHat.icon}
              onChange={(e) => setNewHat({ ...newHat, icon: e.target.value })}
              placeholder="Emoji"
            />
            <Textarea
              label="Description"
              value={newHat.description}
              onChange={(e) => setNewHat({ ...newHat, description: e.target.value })}
              placeholder="What is this hat for?"
              rows={2}
            />
          </div>
        </ModalBody>
        <ModalFooter>
          <Button variant="ghost" onClick={() => setShowCreateModal(false)}>
            Cancel
          </Button>
          <Button
            onClick={() => createMutation.mutate(newHat)}
            isLoading={createMutation.isPending}
            disabled={!newHat.name}
          >
            Create Hat
          </Button>
        </ModalFooter>
      </Modal>
    </>
  )
}

// Appearance Settings
function AppearanceSettings() {
  const toast = useToast()
  const [theme, setTheme] = useState('dark')

  return (
    <Card>
      <CardHeader title="Appearance" subtitle="Customize the look and feel" />
      <CardContent>
        <div className="space-y-4">
          <div>
            <label className="text-sm text-gray-400 mb-2 block">Theme</label>
            <div className="flex gap-3">
              <button
                onClick={() => setTheme('dark')}
                className={clsx(
                  'flex-1 p-4 rounded-lg border text-center transition-all',
                  theme === 'dark'
                    ? 'bg-primary-500/20 border-primary-500/50'
                    : 'bg-surface-light/20 border-surface-border hover:border-gray-600'
                )}
              >
                <div className="text-2xl mb-2">🌙</div>
                <div className="text-sm text-white">Dark</div>
              </button>
              <button
                onClick={() => setTheme('light')}
                disabled
                className="flex-1 p-4 rounded-lg border bg-surface-light/20 border-surface-border opacity-50 cursor-not-allowed text-center"
              >
                <div className="text-2xl mb-2">☀️</div>
                <div className="text-sm text-gray-500">Light (Coming Soon)</div>
              </button>
            </div>
          </div>
        </div>
      </CardContent>
      <CardFooter>
        <Button onClick={() => toast.info('Theme saved', 'Your preference has been updated')}>
          Save Changes
        </Button>
      </CardFooter>
    </Card>
  )
}

// Danger Zone
function DangerZone() {
  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const toast = useToast()

  const handleDeleteAccount = () => {
    toast.error('Action blocked', 'Account deletion is disabled in this version')
    setShowDeleteModal(false)
  }

  return (
    <>
      <Card className="border-red-500/30">
        <CardHeader title="Danger Zone" subtitle="Irreversible actions" />
        <CardContent>
          <div className="space-y-4">
            <div className="flex items-center justify-between p-4 rounded-lg border border-red-500/30 bg-red-500/5">
              <div>
                <div className="font-medium text-white">Reset Learning Data</div>
                <div className="text-sm text-gray-400">
                  Clear all learned patterns and preferences
                </div>
              </div>
              <Button variant="danger" size="sm">
                Reset
              </Button>
            </div>

            <div className="flex items-center justify-between p-4 rounded-lg border border-red-500/30 bg-red-500/5">
              <div>
                <div className="font-medium text-white">Delete Account</div>
                <div className="text-sm text-gray-400">
                  Permanently delete your account and all data
                </div>
              </div>
              <Button variant="danger" size="sm" onClick={() => setShowDeleteModal(true)}>
                Delete
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <ConfirmModal
        isOpen={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        onConfirm={handleDeleteAccount}
        title="Delete Account"
        message="Are you sure you want to delete your account? This action cannot be undone and all your data will be permanently lost."
        confirmText="Delete Account"
        variant="danger"
      />
    </>
  )
}

// Toggle Item Component
function ToggleItem({
  label,
  description,
  checked,
  onChange,
  disabled = false,
}: {
  label: string
  description: string
  checked: boolean
  onChange: (checked: boolean) => void
  disabled?: boolean
}) {
  return (
    <div
      className={clsx(
        'flex items-center justify-between p-4 rounded-lg bg-surface-light/20 border border-surface-border',
        disabled && 'opacity-50'
      )}
    >
      <div>
        <div className="font-medium text-white">{label}</div>
        <div className="text-sm text-gray-400">{description}</div>
      </div>
      <button
        onClick={() => !disabled && onChange(!checked)}
        disabled={disabled}
        className={clsx(
          'w-12 h-6 rounded-full transition-colors relative',
          checked ? 'bg-primary-500' : 'bg-surface-lighter',
          disabled && 'cursor-not-allowed'
        )}
      >
        <span
          className={clsx(
            'absolute top-1 w-4 h-4 rounded-full bg-white transition-transform',
            checked ? 'translate-x-7' : 'translate-x-1'
          )}
        />
      </button>
    </div>
  )
}

export default SettingsPage
