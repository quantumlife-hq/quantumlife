import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { motion, AnimatePresence } from 'framer-motion'
import { clsx } from 'clsx'
import {
  SparklesIcon,
  UserIcon,
  LinkIcon,
  ShieldCheckIcon,
  CheckIcon,
  ArrowRightIcon,
  ArrowLeftIcon,
} from '@heroicons/react/24/outline'
import { api } from '@/services/api'
import { useAppStore } from '@/stores/app'
import {
  Card,
  Button,
  Input,
  Textarea,
  Badge,
  Progress,
  useToast,
} from '@/components/ui'
import type { Provider } from '@/types'

// Onboarding steps
const steps = [
  { id: 'welcome', title: 'Welcome', icon: SparklesIcon },
  { id: 'profile', title: 'Profile', icon: UserIcon },
  { id: 'connections', title: 'Connect', icon: LinkIcon },
  { id: 'trust', title: 'Trust Setup', icon: ShieldCheckIcon },
]

interface OnboardingData {
  name: string
  email: string
  bio: string
  selectedProviders: string[]
  trustPreferences: {
    defaultMode: 'ask' | 'suggest' | 'auto'
    notifications: boolean
  }
}

export function OnboardingPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const toast = useToast()
  const { setIdentity, setSettings } = useAppStore()

  const [currentStep, setCurrentStep] = useState(0)
  const [data, setData] = useState<OnboardingData>({
    name: '',
    email: '',
    bio: '',
    selectedProviders: [],
    trustPreferences: {
      defaultMode: 'suggest',
      notifications: true,
    },
  })

  // Save identity mutation
  const saveIdentityMutation = useMutation({
    mutationFn: (identity: { name: string; email: string; bio?: string }) =>
      api.identity.update(identity),
    onSuccess: (savedIdentity) => {
      setIdentity(savedIdentity)
    },
  })

  // Complete onboarding
  const completeOnboarding = async () => {
    try {
      // Save identity
      await saveIdentityMutation.mutateAsync({
        name: data.name,
        email: data.email,
        bio: data.bio,
      })

      // Save settings
      await api.settings.update({
        autonomyMode: data.trustPreferences.defaultMode,
        notifications: data.trustPreferences.notifications,
        onboarded: true,
      })

      // Update local state
      setSettings({
        autonomyMode: data.trustPreferences.defaultMode,
        notifications: data.trustPreferences.notifications,
        onboarded: true,
      })

      // Invalidate queries
      queryClient.invalidateQueries()

      toast.success('Welcome to QuantumLife!', 'Your account has been set up successfully')
      navigate('/')
    } catch (error) {
      toast.error('Setup failed', 'Please try again')
    }
  }

  const nextStep = () => {
    if (currentStep < steps.length - 1) {
      setCurrentStep(currentStep + 1)
    } else {
      completeOnboarding()
    }
  }

  const prevStep = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1)
    }
  }

  const canProceed = () => {
    switch (currentStep) {
      case 0:
        return true // Welcome step
      case 1:
        return data.name.trim().length >= 2 && data.email.includes('@')
      case 2:
        return true // Connections are optional
      case 3:
        return true // Trust preferences have defaults
      default:
        return false
    }
  }

  return (
    <div className="min-h-screen gradient-bg flex items-center justify-center p-4">
      <div className="w-full max-w-2xl">
        {/* Logo */}
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-center mb-8"
        >
          <div className="w-16 h-16 mx-auto rounded-2xl bg-gradient-to-br from-primary-500 to-violet-600 flex items-center justify-center shadow-glow-sm">
            <SparklesIcon className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-white mt-4">QuantumLife</h1>
          <p className="text-gray-400 mt-1">Your AI-powered digital life manager</p>
        </motion.div>

        {/* Progress */}
        <div className="mb-8">
          <Progress
            value={(currentStep + 1) / steps.length * 100}
            variant="gradient"
            size="sm"
          />
          <div className="flex justify-between mt-4">
            {steps.map((step, index) => (
              <div
                key={step.id}
                className={clsx(
                  'flex items-center gap-2 text-sm',
                  index <= currentStep ? 'text-white' : 'text-gray-600'
                )}
              >
                <div
                  className={clsx(
                    'w-6 h-6 rounded-full flex items-center justify-center text-xs',
                    index < currentStep
                      ? 'bg-primary-500 text-white'
                      : index === currentStep
                      ? 'bg-primary-500/30 text-primary-300 border border-primary-500'
                      : 'bg-surface-lighter text-gray-500'
                  )}
                >
                  {index < currentStep ? (
                    <CheckIcon className="w-3 h-3" />
                  ) : (
                    index + 1
                  )}
                </div>
                <span className="hidden sm:inline">{step.title}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Step Content */}
        <Card className="overflow-hidden">
          <AnimatePresence mode="wait">
            <motion.div
              key={currentStep}
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
              transition={{ duration: 0.2 }}
              className="p-6"
            >
              {currentStep === 0 && <WelcomeStep />}
              {currentStep === 1 && (
                <ProfileStep data={data} onChange={setData} />
              )}
              {currentStep === 2 && (
                <ConnectionsStep data={data} onChange={setData} />
              )}
              {currentStep === 3 && (
                <TrustStep data={data} onChange={setData} />
              )}
            </motion.div>
          </AnimatePresence>

          {/* Navigation */}
          <div className="flex items-center justify-between p-6 border-t border-surface-border">
            <Button
              variant="ghost"
              onClick={prevStep}
              disabled={currentStep === 0}
            >
              <ArrowLeftIcon className="w-4 h-4 mr-2" />
              Back
            </Button>
            <Button
              onClick={nextStep}
              disabled={!canProceed()}
              isLoading={saveIdentityMutation.isPending}
            >
              {currentStep === steps.length - 1 ? 'Complete Setup' : 'Continue'}
              {currentStep < steps.length - 1 && (
                <ArrowRightIcon className="w-4 h-4 ml-2" />
              )}
            </Button>
          </div>
        </Card>
      </div>
    </div>
  )
}

// Welcome Step
function WelcomeStep() {
  return (
    <div className="text-center py-8">
      <div className="w-24 h-24 mx-auto rounded-full bg-gradient-to-br from-primary-500/20 to-violet-500/20 flex items-center justify-center mb-6">
        <SparklesIcon className="w-12 h-12 text-primary-400" />
      </div>
      <h2 className="text-2xl font-bold text-white mb-4">
        Welcome to QuantumLife
      </h2>
      <p className="text-gray-400 max-w-md mx-auto mb-8">
        Your intelligent digital life manager. Connect your services, set your preferences,
        and let AI help you stay organized and productive.
      </p>
      <div className="grid grid-cols-3 gap-4 max-w-sm mx-auto">
        <FeatureCard icon="🔗" label="Connect Services" />
        <FeatureCard icon="🤖" label="AI Assistance" />
        <FeatureCard icon="🛡️" label="You're in Control" />
      </div>
    </div>
  )
}

function FeatureCard({ icon, label }: { icon: string; label: string }) {
  return (
    <div className="text-center p-3 rounded-lg bg-surface-light/20">
      <span className="text-2xl block mb-2">{icon}</span>
      <span className="text-xs text-gray-400">{label}</span>
    </div>
  )
}

// Profile Step
function ProfileStep({
  data,
  onChange,
}: {
  data: OnboardingData
  onChange: (data: OnboardingData) => void
}) {
  return (
    <div>
      <h2 className="text-xl font-bold text-white mb-2">Create your profile</h2>
      <p className="text-gray-400 mb-6">Tell us a bit about yourself</p>

      <div className="space-y-4">
        <Input
          label="Your name"
          placeholder="Enter your full name"
          value={data.name}
          onChange={(e) => onChange({ ...data, name: e.target.value })}
        />
        <Input
          label="Email address"
          type="email"
          placeholder="you@example.com"
          value={data.email}
          onChange={(e) => onChange({ ...data, email: e.target.value })}
        />
        <Textarea
          label="Bio (optional)"
          placeholder="A brief description about yourself..."
          value={data.bio}
          onChange={(e) => onChange({ ...data, bio: e.target.value })}
          rows={3}
        />
      </div>
    </div>
  )
}

// Connections Step
function ConnectionsStep({
  data,
  onChange,
}: {
  data: OnboardingData
  onChange: (data: OnboardingData) => void
}) {
  const popularProviders = [
    { key: 'google-mail', name: 'Gmail', icon: '📧', category: 'email' },
    { key: 'google-calendar', name: 'Google Calendar', icon: '📅', category: 'calendar' },
    { key: 'slack', name: 'Slack', icon: '💬', category: 'communication' },
    { key: 'notion', name: 'Notion', icon: '📝', category: 'productivity' },
    { key: 'github', name: 'GitHub', icon: '💻', category: 'development' },
    { key: 'linear', name: 'Linear', icon: '📋', category: 'productivity' },
  ]

  const toggleProvider = (key: string) => {
    const selected = data.selectedProviders.includes(key)
      ? data.selectedProviders.filter((p) => p !== key)
      : [...data.selectedProviders, key]
    onChange({ ...data, selectedProviders: selected })
  }

  return (
    <div>
      <h2 className="text-xl font-bold text-white mb-2">Connect your services</h2>
      <p className="text-gray-400 mb-6">
        Select services you'd like to connect. You can add more later.
      </p>

      <div className="grid grid-cols-2 gap-3">
        {popularProviders.map((provider) => (
          <button
            key={provider.key}
            onClick={() => toggleProvider(provider.key)}
            className={clsx(
              'flex items-center gap-3 p-4 rounded-xl border transition-all text-left',
              data.selectedProviders.includes(provider.key)
                ? 'bg-primary-500/20 border-primary-500/50'
                : 'bg-surface-light/20 border-surface-border hover:border-gray-600'
            )}
          >
            <span className="text-2xl">{provider.icon}</span>
            <div className="flex-1">
              <div className="font-medium text-white">{provider.name}</div>
              <div className="text-xs text-gray-500">{provider.category}</div>
            </div>
            {data.selectedProviders.includes(provider.key) && (
              <CheckIcon className="w-5 h-5 text-primary-400" />
            )}
          </button>
        ))}
      </div>

      <p className="text-sm text-gray-500 mt-4 text-center">
        You can connect over 500+ services from the Connections page
      </p>
    </div>
  )
}

// Trust Step
function TrustStep({
  data,
  onChange,
}: {
  data: OnboardingData
  onChange: (data: OnboardingData) => void
}) {
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
      recommended: true,
    },
    {
      id: 'auto',
      name: 'Auto-pilot',
      description: 'AI acts autonomously within your trust limits',
      icon: '🚀',
    },
  ]

  return (
    <div>
      <h2 className="text-xl font-bold text-white mb-2">Set your trust preferences</h2>
      <p className="text-gray-400 mb-6">
        Choose how much autonomy to give your AI assistant
      </p>

      <div className="space-y-3 mb-6">
        {modes.map((mode) => (
          <button
            key={mode.id}
            onClick={() =>
              onChange({
                ...data,
                trustPreferences: {
                  ...data.trustPreferences,
                  defaultMode: mode.id as 'ask' | 'suggest' | 'auto',
                },
              })
            }
            className={clsx(
              'w-full flex items-start gap-4 p-4 rounded-xl border transition-all text-left',
              data.trustPreferences.defaultMode === mode.id
                ? 'bg-primary-500/20 border-primary-500/50'
                : 'bg-surface-light/20 border-surface-border hover:border-gray-600'
            )}
          >
            <span className="text-2xl">{mode.icon}</span>
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <span className="font-medium text-white">{mode.name}</span>
                {mode.recommended && (
                  <Badge variant="primary" size="sm">
                    Recommended
                  </Badge>
                )}
              </div>
              <p className="text-sm text-gray-400 mt-1">{mode.description}</p>
            </div>
            {data.trustPreferences.defaultMode === mode.id && (
              <CheckIcon className="w-5 h-5 text-primary-400 flex-shrink-0" />
            )}
          </button>
        ))}
      </div>

      {/* Notifications toggle */}
      <div className="flex items-center justify-between p-4 rounded-xl bg-surface-light/20 border border-surface-border">
        <div>
          <div className="font-medium text-white">Enable notifications</div>
          <div className="text-sm text-gray-400">Get notified about AI suggestions</div>
        </div>
        <button
          onClick={() =>
            onChange({
              ...data,
              trustPreferences: {
                ...data.trustPreferences,
                notifications: !data.trustPreferences.notifications,
              },
            })
          }
          className={clsx(
            'w-12 h-6 rounded-full transition-colors relative',
            data.trustPreferences.notifications
              ? 'bg-primary-500'
              : 'bg-surface-lighter'
          )}
        >
          <span
            className={clsx(
              'absolute top-1 w-4 h-4 rounded-full bg-white transition-transform',
              data.trustPreferences.notifications ? 'translate-x-7' : 'translate-x-1'
            )}
          />
        </button>
      </div>
    </div>
  )
}

export default OnboardingPage
