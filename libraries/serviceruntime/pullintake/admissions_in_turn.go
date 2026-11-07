package pullintake

import "context"

type admissionsInTurn []Admission

func AdmissionOfEach(admissions ...Admission) Admission {
	return admissionsInTurn(admissions)
}

func (a admissionsInTurn) AwaitAdmission(ctx context.Context) error {
	for _, admission := range a {
		if admission == nil {
			continue
		}
		if err := admission.AwaitAdmission(ctx); err != nil {
			return err
		}
	}
	return nil
}
