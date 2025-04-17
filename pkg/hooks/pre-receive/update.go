package pre_receive

import "fmt"

// Update refreshes the pre-receive hook, either locally or globally.
func Update(global bool) error {
	if global {
		return updateGlobal()
	}
	return updateLocal()
}

// updateLocal updates the local pre-receive hook.
func updateLocal() error {
	fmt.Println("Updating local pre-receive hook...")

	if err := uninstallLocal(); err != nil {
		return fmt.Errorf("failed to uninstall existing local pre-receive hook: %v", err)
	}

	if err := installLocal(); err != nil {
		return fmt.Errorf("failed to install new local pre-receive hook: %v", err)
	}

	fmt.Println("Local pre-receive hook updated successfully.")
	return nil
}

// updateGlobal updates the global pre-receive hook.
func updateGlobal() error {
	fmt.Println("Updating global pre-receive hook...")

	if err := uninstallGlobal(); err != nil {
		return fmt.Errorf("failed to uninstall existing global pre-receive hook: %v", err)
	}

	if err := installGlobal(); err != nil {
		return fmt.Errorf("failed to install new global pre-receive hook: %v", err)
	}

	fmt.Println("Global pre-receive hook updated successfully.")
	return nil
}
