export function Settings() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">
          Manage your QuantumLife preferences
        </p>
      </div>

      <div className="rounded-lg border bg-card p-6">
        <p className="text-muted-foreground">
          Settings page coming soon. This will include:
        </p>
        <ul className="mt-4 space-y-2 text-sm text-muted-foreground list-disc list-inside">
          <li>Autonomy mode preferences</li>
          <li>Notification settings</li>
          <li>Connected spaces management</li>
          <li>Data export options</li>
          <li>Account settings</li>
        </ul>
      </div>
    </div>
  );
}
