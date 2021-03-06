package tests

import (
	"testing"
)

//TODO Test full restore - how to delete data directory?

func TestBackrestDeltaRestore(t *testing.T) {
	t.Parallel()
	t.Log("Testing the 'backrest-restore' example...")

	harness := setup(t, timeout, true)
	env := []string{}
	_, err := harness.runExample("examples/kube/backrest/run.sh", env, t)
	if err != nil {
		t.Fatalf("Could not run example: %s", err)
	}
	if harness.Cleanup {
		defer harness.Client.DeleteNamespace(harness.Namespace)
		defer harness.runExample("examples/kube/backrest/cleanup.sh", env, t)
	}

	t.Log("Checking if pods are ready to use...")
	pods := []string{"backrest"}
	if err := harness.Client.CheckPods(harness.Namespace, pods); err != nil {
		t.Fatal(err)
	}

	t.Log("Running full backup...")
	// Required for OCP - backrest gets confused when random UIDs aren't found in PAM.
	// Exec doesn't load bashrc or bash_profile, so we need to set this explicitly.
	nsswrapper := []string{"env", "LD_PRELOAD=/usr/lib64/libnss_wrapper.so", "NSS_WRAPPER_PASSWD=/tmp/passwd", "NSS_WRAPPER_GROUP=/tmp/group"}
	fullBackup := []string{"/usr/bin/pgbackrest", "--stanza=db", "backup", "--type=full"}
	cmd := append(nsswrapper, fullBackup...)
	_, stderr, err := harness.Client.Exec(harness.Namespace, "backrest", "backrest", cmd)
	if err != nil {
		t.Logf("\n%s", stderr)
		t.Fatalf("Error execing into container: %s", err)
	}

	// Cleanup of pod and service
	if err := harness.Client.DeletePod(harness.Namespace, "backrest"); err != nil {
		t.Fatal(err)
	}

	if !harness.Client.IsPodDeleted(harness.Namespace, "backrest") {
		t.Fatal("The 'backrest' pod didn't delete (it should have)")
	}

	if err := harness.Client.DeleteService(harness.Namespace, "backrest"); err != nil {
		t.Fatal(err)
	}

	// Delta Restore
	_, err = harness.runExample("examples/kube/backrest-restore/delta-restore.sh", env, t)
	if err != nil {
		t.Fatalf("Could not run example: %s", err)
	}
	if harness.Cleanup {
		defer harness.runExample("examples/kube/backrest-restore/cleanup.sh", env, t)
	}

	if ok, err := harness.Client.IsJobComplete(harness.Namespace, "backrest-delta-restore-job"); !ok {
		t.Fatal(err)
	}

	_, err = harness.runExample("examples/kube/backrest/post-restore.sh", env, t)
	if err != nil {
		t.Fatalf("Could not run 'post-restore.sh': %s", err)
	}

	t.Log("Checking if pods are ready to use...")
	if err := harness.Client.CheckPods(harness.Namespace, pods); err != nil {
		t.Fatal(err)
	}

	report, err := harness.createReport()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(report)
}
