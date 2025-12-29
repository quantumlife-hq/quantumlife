import { Fragment, type ReactNode } from 'react'
import { Dialog, Transition, TransitionChild, DialogPanel, DialogTitle } from '@headlessui/react'
import { clsx } from 'clsx'
import { XMarkIcon } from '@heroicons/react/24/outline'
import { IconButton } from './Button'

export interface ModalProps {
  isOpen: boolean
  onClose: () => void
  title?: string
  description?: string
  children: ReactNode
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  showClose?: boolean
  closeOnOverlayClick?: boolean
}

const sizeStyles = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
  full: 'max-w-4xl',
}

export function Modal({
  isOpen,
  onClose,
  title,
  description,
  children,
  size = 'md',
  showClose = true,
  closeOnOverlayClick = true,
}: ModalProps) {
  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog
        as="div"
        className="relative z-50"
        onClose={closeOnOverlayClick ? onClose : () => {}}
      >
        {/* Backdrop */}
        <TransitionChild
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/60 backdrop-blur-sm" />
        </TransitionChild>

        {/* Modal container */}
        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4">
            <TransitionChild
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <DialogPanel
                className={clsx(
                  'w-full transform overflow-hidden rounded-2xl',
                  'glass-dark p-6',
                  'shadow-2xl transition-all',
                  sizeStyles[size]
                )}
              >
                {/* Header */}
                {(title || showClose) && (
                  <div className="flex items-start justify-between gap-4 mb-4">
                    <div>
                      {title && (
                        <DialogTitle as="h3" className="text-lg font-semibold text-white">
                          {title}
                        </DialogTitle>
                      )}
                      {description && (
                        <p className="mt-1 text-sm text-gray-400">{description}</p>
                      )}
                    </div>
                    {showClose && (
                      <IconButton
                        icon={<XMarkIcon className="w-5 h-5" />}
                        aria-label="Close modal"
                        variant="ghost"
                        size="sm"
                        onClick={onClose}
                      />
                    )}
                  </div>
                )}

                {/* Content */}
                {children}
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </Transition>
  )
}

// Modal subcomponents for structured content
export function ModalBody({ className, children }: { className?: string; children: ReactNode }) {
  return <div className={clsx('', className)}>{children}</div>
}

export function ModalFooter({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <div className={clsx('flex items-center justify-end gap-3 mt-6 pt-4 border-t border-surface-border', className)}>
      {children}
    </div>
  )
}

// Confirmation Modal
export interface ConfirmModalProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: 'primary' | 'danger'
  isLoading?: boolean
}

export function ConfirmModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  variant = 'primary',
  isLoading = false,
}: ConfirmModalProps) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} size="sm">
      <ModalBody>
        <p className="text-gray-300">{message}</p>
      </ModalBody>
      <ModalFooter>
        <button
          type="button"
          className="btn btn-sm text-gray-400 hover:text-white"
          onClick={onClose}
          disabled={isLoading}
        >
          {cancelText}
        </button>
        <button
          type="button"
          className={clsx(
            'btn btn-sm',
            variant === 'danger' ? 'bg-red-600 hover:bg-red-500 text-white' : 'btn-primary'
          )}
          onClick={onConfirm}
          disabled={isLoading}
        >
          {isLoading ? 'Loading...' : confirmText}
        </button>
      </ModalFooter>
    </Modal>
  )
}
