package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func newID() string { return uuid.New().String() }

func hashPassword(pw string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	if err != nil {
		log.Fatalf("bcrypt: %v", err)
	}
	return string(h)
}

func daysAgo(d int) time.Time {
	return time.Now().UTC().AddDate(0, 0, -d)
}

func randBetween(min, max int) int {
	return min + rand.Intn(max-min+1)
}

// ── main ─────────────────────────────────────────────────────────────────────

func main() {
	_ = godotenv.Load()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOr("DB_HOST", "localhost"),
		envOr("DB_PORT", "5432"),
		envOr("DB_USER", "jobber"),
		envOr("DB_PASSWORD", "jobber"),
		envOr("DB_NAME", "jobber"),
		envOr("DB_SSL_MODE", "disable"),
	)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}
	fmt.Println("connected to database")

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	// ── clean up previous seed data ──────────────────────────────────────
	const seedEmail = "seed@jobber.dev"
	_, _ = tx.Exec(ctx, `DELETE FROM users WHERE email = $1`, seedEmail)
	fmt.Println("cleaned previous seed data")

	// ── 1. user ──────────────────────────────────────────────────────────
	userID := newID()
	now := time.Now().UTC()
	createdAt := daysAgo(120) // account created ~4 months ago

	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, email, name, password_hash, locale, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, seedEmail, "Alex Jobseeker", hashPassword("password123"), "en", createdAt, createdAt,
	)
	must(err, "create user")
	fmt.Printf("created user: %s / password123\n", seedEmail)

	// ── 1b. subscription (pro) ──────────────────────────────────────────
	_, err = tx.Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, status, plan, created_at, updated_at)
		 VALUES ($1, $2, 'active', 'pro', $3, $3)
		 ON CONFLICT (user_id) DO UPDATE SET plan = 'pro', status = 'active', updated_at = $3`,
		newID(), userID, createdAt,
	)
	must(err, "create subscription")
	fmt.Println("created pro subscription")

	// ── 1c. resume builders (4 different templates) ─────────────────────
	type rbDef struct {
		id, title, templateID, font, color, layout string
		spacing, marginTop, marginBot, marginL, marginR, sidebarW int
	}
	resumeBuilders := []rbDef{
		{
			newID(), "Senior Software Engineer", "00000000-0000-0000-0000-000000000001",
			"Georgia", "#1e40af", "single", 100, 40, 40, 40, 40, 35,
		},
		{
			newID(), "Frontend Developer", "00000000-0000-0000-0000-000000000005",
			"Inter", "#e11d48", "double-left", 90, 36, 36, 36, 36, 32,
		},
		{
			newID(), "Full-Stack Engineer", "00000000-0000-0000-0000-000000000009",
			"Roboto", "#7c3aed", "single", 100, 40, 40, 40, 40, 35,
		},
		{
			newID(), "DevOps & Cloud Resume", "00000000-0000-0000-0000-00000000000c",
			"Open Sans", "#059669", "double-right", 95, 32, 32, 32, 32, 38,
		},
	}

	type rbContact struct {
		fullName, email, phone, location, website, linkedin, github string
	}
	contacts := []rbContact{
		{"Alex Jobseeker", "alex@example.com", "+1 (555) 123-4567", "San Francisco, CA", "alexdev.io", "linkedin.com/in/alexjobseeker", "github.com/alexjob"},
		{"Alex Jobseeker", "alex@example.com", "+1 (555) 123-4567", "San Francisco, CA", "alexdev.io", "linkedin.com/in/alexjobseeker", "github.com/alexjob"},
		{"Alex Jobseeker", "alex@example.com", "+1 (555) 123-4567", "San Francisco, CA", "alexdev.io", "linkedin.com/in/alexjobseeker", "github.com/alexjob"},
		{"Alex Jobseeker", "alex@example.com", "+1 (555) 123-4567", "San Francisco, CA", "", "linkedin.com/in/alexjobseeker", "github.com/alexjob"},
	}

	summaries := []string{
		"Results-driven senior software engineer with 8+ years of experience building scalable distributed systems. Proven track record of leading cross-functional teams and delivering high-impact products at top-tier tech companies. Passionate about clean architecture, performance optimization, and mentoring junior engineers.",
		"Creative frontend developer with 5 years of experience crafting beautiful, accessible, and performant user interfaces. Specialized in React, TypeScript, and modern CSS. Strong eye for design with experience collaborating closely with UX teams to deliver pixel-perfect implementations.",
		"Versatile full-stack engineer with 6 years of experience across the entire web stack. Proficient in Go, TypeScript, React, and PostgreSQL. Experienced in building and deploying microservices, REST APIs, and real-time applications. Comfortable wearing many hats in fast-paced startup environments.",
		"DevOps engineer with 7 years of experience designing and maintaining cloud infrastructure at scale. Expert in AWS, Kubernetes, Terraform, and CI/CD pipelines. Passionate about reliability engineering, infrastructure as code, and enabling development teams to ship faster with confidence.",
	}

	type expDef struct {
		company, position, location, startDate, endDate, description string
		isCurrent                                                    bool
	}
	experienceSets := [][]expDef{
		{
			{"TechNova Inc.", "Senior Software Engineer", "San Francisco, CA", "2021-03", "", "Led the redesign of the core data processing pipeline, improving throughput by 40%. Mentored a team of 4 junior engineers. Architected a new event-driven microservices system handling 50K+ events/second.", true},
			{"CloudScale Systems", "Software Engineer", "Remote", "2018-06", "2021-02", "Built and maintained Go microservices for the cloud orchestration platform. Implemented distributed caching layer reducing API latency by 60%. Contributed to open-source Kubernetes operators.", false},
			{"DataPulse Analytics", "Junior Software Engineer", "New York, NY", "2016-08", "2018-05", "Developed RESTful APIs using Python/Django for the analytics dashboard. Wrote comprehensive test suites achieving 90%+ code coverage. Participated in on-call rotation for production systems.", false},
		},
		{
			{"PixelCraft Studios", "Senior Frontend Developer", "Los Angeles, CA", "2022-01", "", "Lead frontend architect for the design collaboration platform serving 100K+ users. Built a custom component library with 50+ accessible React components. Implemented real-time collaborative editing using WebSocket.", true},
			{"DataPulse Analytics", "Frontend Developer", "New York, NY", "2019-09", "2021-12", "Developed the analytics dashboard using React and D3.js with complex data visualizations. Improved Lighthouse performance score from 45 to 92. Built responsive layouts supporting mobile, tablet, and desktop.", false},
		},
		{
			{"GreenByte Solutions", "Full-Stack Engineer", "Austin, TX", "2020-06", "", "Building sustainability tracking platform with Go backend and React frontend. Designed PostgreSQL schema handling 10M+ daily measurements. Implemented real-time WebSocket dashboards for IoT sensor data.", true},
			{"FinEdge", "Software Developer", "Chicago, IL", "2018-01", "2020-05", "Developed payment processing microservices handling $2M+ daily transactions. Built React-based admin portal for transaction monitoring. Implemented end-to-end encryption for sensitive financial data.", false},
		},
		{
			{"Quantum Labs", "Senior DevOps Engineer", "Seattle, WA", "2021-07", "", "Managing Kubernetes clusters across 3 AWS regions serving 5M+ daily active users. Built Terraform modules reducing infrastructure provisioning time from days to minutes. Designed zero-downtime deployment pipeline with automated canary releases.", true},
			{"InfraCore", "DevOps Engineer", "Denver, CO", "2019-03", "2021-06", "Migrated legacy monolith to containerized microservices on EKS. Implemented GitOps workflow with ArgoCD reducing deployment errors by 80%. Built comprehensive monitoring stack with Prometheus, Grafana, and PagerDuty.", false},
			{"CloudScale Systems", "Junior SRE", "Remote", "2017-05", "2019-02", "Maintained 99.95% uptime SLA for production services. Automated incident response playbooks reducing MTTR by 50%. Managed CI/CD pipelines using Jenkins and GitHub Actions.", false},
		},
	}

	type eduDef struct {
		institution, degree, field, startDate, endDate, gpa string
	}
	educationSets := [][]eduDef{
		{{"Stanford University", "M.S.", "Computer Science", "2014-09", "2016-06", "3.9"}, {"UC Berkeley", "B.S.", "Computer Science", "2010-09", "2014-06", "3.7"}},
		{{"Rhode Island School of Design", "B.F.A.", "Graphic Design", "2015-09", "2019-06", "3.8"}},
		{{"University of Texas at Austin", "B.S.", "Computer Engineering", "2014-09", "2018-05", "3.6"}},
		{{"Georgia Tech", "M.S.", "Computer Science", "2013-09", "2015-05", "3.8"}, {"University of Washington", "B.S.", "Information Systems", "2009-09", "2013-06", "3.5"}},
	}

	type skillDef struct{ name, level string }
	skillSets := [][]skillDef{
		{{"Go", "Expert"}, {"Python", "Advanced"}, {"PostgreSQL", "Expert"}, {"Kubernetes", "Advanced"}, {"gRPC", "Advanced"}, {"Redis", "Advanced"}, {"Docker", "Expert"}, {"AWS", "Advanced"}},
		{{"React", "Expert"}, {"TypeScript", "Expert"}, {"CSS/Tailwind", "Expert"}, {"Next.js", "Advanced"}, {"Figma", "Advanced"}, {"Storybook", "Advanced"}, {"Vitest", "Advanced"}, {"Accessibility", "Advanced"}},
		{{"Go", "Advanced"}, {"TypeScript", "Advanced"}, {"React", "Advanced"}, {"PostgreSQL", "Advanced"}, {"Docker", "Intermediate"}, {"GraphQL", "Intermediate"}, {"Redis", "Intermediate"}, {"Node.js", "Advanced"}},
		{{"Kubernetes", "Expert"}, {"Terraform", "Expert"}, {"AWS", "Expert"}, {"Docker", "Expert"}, {"GitHub Actions", "Expert"}, {"Prometheus", "Advanced"}, {"ArgoCD", "Advanced"}, {"Linux", "Expert"}},
	}

	type langDef struct{ name, proficiency string }
	langSets := [][]langDef{
		{{"English", "Native"}, {"Spanish", "Intermediate"}},
		{{"English", "Native"}, {"French", "Conversational"}},
		{{"English", "Native"}, {"German", "Intermediate"}, {"Ukrainian", "Native"}},
		{{"English", "Native"}, {"Japanese", "Conversational"}},
	}

	type certDef struct{ name, issuer, issueDate, url string }
	certSets := [][]certDef{
		{{"AWS Solutions Architect - Professional", "Amazon Web Services", "2023-04", "https://aws.amazon.com/certification/"}},
		{},
		{{"Google Cloud Professional Developer", "Google", "2022-11", "https://cloud.google.com/certification"}},
		{{"CKA: Certified Kubernetes Administrator", "CNCF", "2023-01", "https://www.cncf.io/certification/cka/"}, {"AWS DevOps Engineer - Professional", "Amazon Web Services", "2022-08", "https://aws.amazon.com/certification/"}},
	}

	sectionKeys := []string{"summary", "experience", "education", "skills", "languages", "certifications", "projects", "volunteering", "custom"}

	for i, rb := range resumeBuilders {
		rbCreated := daysAgo(randBetween(30, 90))
		_, err = tx.Exec(ctx,
			`INSERT INTO resume_builders (id, user_id, title, template_id, font_family, primary_color, spacing, margin_top, margin_bottom, margin_left, margin_right, layout_mode, sidebar_width, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)`,
			rb.id, userID, rb.title, rb.templateID, rb.font, rb.color, rb.spacing, rb.marginTop, rb.marginBot, rb.marginL, rb.marginR, rb.layout, rb.sidebarW, rbCreated,
		)
		must(err, "create resume builder "+rb.title)

		// Contact
		c := contacts[i]
		_, err = tx.Exec(ctx,
			`INSERT INTO resume_contacts (id, resume_builder_id, full_name, email, phone, location, website, linkedin, github)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			newID(), rb.id, c.fullName, c.email, c.phone, c.location, c.website, c.linkedin, c.github,
		)
		must(err, "create contact for "+rb.title)

		// Summary
		_, err = tx.Exec(ctx,
			`INSERT INTO resume_summaries (id, resume_builder_id, content) VALUES ($1, $2, $3)`,
			newID(), rb.id, summaries[i],
		)
		must(err, "create summary for "+rb.title)

		// Experiences
		for j, exp := range experienceSets[i] {
			_, err = tx.Exec(ctx,
				`INSERT INTO resume_experiences (id, resume_builder_id, company, position, location, start_date, end_date, is_current, description, sort_order)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				newID(), rb.id, exp.company, exp.position, exp.location, exp.startDate, exp.endDate, exp.isCurrent, exp.description, j,
			)
			must(err, "create experience for "+rb.title)
		}

		// Educations
		for j, edu := range educationSets[i] {
			_, err = tx.Exec(ctx,
				`INSERT INTO resume_educations (id, resume_builder_id, institution, degree, field_of_study, start_date, end_date, gpa, sort_order)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				newID(), rb.id, edu.institution, edu.degree, edu.field, edu.startDate, edu.endDate, edu.gpa, j,
			)
			must(err, "create education for "+rb.title)
		}

		// Skills
		for j, sk := range skillSets[i] {
			_, err = tx.Exec(ctx,
				`INSERT INTO resume_skills (id, resume_builder_id, name, level, sort_order) VALUES ($1, $2, $3, $4, $5)`,
				newID(), rb.id, sk.name, sk.level, j,
			)
			must(err, "create skill for "+rb.title)
		}

		// Languages
		for j, lang := range langSets[i] {
			_, err = tx.Exec(ctx,
				`INSERT INTO resume_languages (id, resume_builder_id, name, proficiency, sort_order) VALUES ($1, $2, $3, $4, $5)`,
				newID(), rb.id, lang.name, lang.proficiency, j,
			)
			must(err, "create language for "+rb.title)
		}

		// Certifications
		for j, cert := range certSets[i] {
			_, err = tx.Exec(ctx,
				`INSERT INTO resume_certifications (id, resume_builder_id, name, issuer, issue_date, url, sort_order)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				newID(), rb.id, cert.name, cert.issuer, cert.issueDate, cert.url, j,
			)
			must(err, "create certification for "+rb.title)
		}

		// Section order
		for j, key := range sectionKeys {
			_, err = tx.Exec(ctx,
				`INSERT INTO resume_section_orders (id, resume_builder_id, section_key, sort_order, is_visible, column_placement)
				 VALUES ($1, $2, $3, $4, true, 'main')`,
				newID(), rb.id, key, j,
			)
			must(err, "create section order for "+rb.title)
		}
	}
	fmt.Printf("created %d resume builders with full content\n", len(resumeBuilders))

	// ── 2. resumes ───────────────────────────────────────────────────────
	type resume struct{ id, title string }
	resumes := []resume{
		{newID(), "Software Engineer Resume"},
		{newID(), "Frontend Developer Resume"},
		{newID(), "Full-Stack Developer Resume"},
	}
	for _, r := range resumes {
		_, err = tx.Exec(ctx,
			`INSERT INTO resumes (id, user_id, title, file_url, storage_type, storage_key, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, NULL, 'external', NULL, true, $4, $4)`,
			r.id, userID, r.title, daysAgo(randBetween(100, 115)),
		)
		must(err, "create resume "+r.title)
	}
	fmt.Printf("created %d resumes\n", len(resumes))

	// ── 3. stage templates ───────────────────────────────────────────────
	type stageTempl struct{ id, name string; order int }
	stages := []stageTempl{
		{newID(), "Applied", 1},
		{newID(), "Screening", 2},
		{newID(), "Technical Interview", 3},
		{newID(), "Take-Home Assignment", 4},
		{newID(), "Final Interview", 5},
		{newID(), "Offer", 6},
	}
	for _, s := range stages {
		_, err = tx.Exec(ctx,
			`INSERT INTO stage_templates (id, user_id, name, "order", created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $5)`,
			s.id, userID, s.name, s.order, daysAgo(115),
		)
		must(err, "create stage template "+s.name)
	}
	fmt.Printf("created %d stage templates\n", len(stages))

	// ── 4. companies ─────────────────────────────────────────────────────
	type company struct{ id, name, location, notes string }
	companies := []company{
		{newID(), "TechNova", "San Francisco, CA", "Series B startup, strong engineering culture"},
		{newID(), "CloudScale Inc.", "Remote", "Cloud infrastructure company, competitive salary"},
		{newID(), "DataPulse", "New York, NY", "Data analytics platform, fast-growing"},
		{newID(), "GreenByte Solutions", "Austin, TX", "Sustainability-focused tech, good WLB"},
		{newID(), "Quantum Labs", "Seattle, WA", "R&D heavy, cutting edge ML work"},
		{newID(), "FinEdge", "Chicago, IL", "Fintech startup, pre-IPO"},
		{newID(), "PixelCraft Studios", "Los Angeles, CA", "Creative tools for designers"},
		{newID(), "InfraCore", "Denver, CO", "DevOps / platform engineering focus"},
	}
	for _, c := range companies {
		_, err = tx.Exec(ctx,
			`INSERT INTO companies (id, user_id, name, location, notes, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
			c.id, userID, c.name, c.location, c.notes, daysAgo(randBetween(90, 110)),
		)
		must(err, "create company "+c.name)
	}
	fmt.Printf("created %d companies\n", len(companies))

	// ── 5. tags ──────────────────────────────────────────────────────────
	type tag struct{ id, name, color string }
	tags := []tag{
		{newID(), "remote", "#3B82F6"},
		{newID(), "onsite", "#F59E0B"},
		{newID(), "hybrid", "#8B5CF6"},
		{newID(), "high-priority", "#EF4444"},
		{newID(), "FAANG-level", "#10B981"},
		{newID(), "startup", "#F97316"},
		{newID(), "referral", "#06B6D4"},
		{newID(), "interesting-tech", "#EC4899"},
		{newID(), "good-comp", "#84CC16"},
		{newID(), "backup", "#6B7280"},
	}
	for _, t := range tags {
		_, err = tx.Exec(ctx,
			`INSERT INTO tags (id, user_id, name, color, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			t.id, userID, t.name, t.color, daysAgo(110),
		)
		must(err, "create tag "+t.name)
	}
	fmt.Printf("created %d tags\n", len(tags))

	// ── 6. jobs ──────────────────────────────────────────────────────────
	type job struct {
		id, companyID, title, source, url, notes, status string
		daysAgo                                           int
	}

	jobs := []job{
		{newID(), companies[0].id, "Senior Software Engineer", "LinkedIn", "https://linkedin.com/jobs/1001", "Exciting ML team", "active", 85},
		{newID(), companies[0].id, "Staff Engineer - Platform", "Company Website", "https://technova.io/careers/staff", "Platform team, high impact", "active", 60},
		{newID(), companies[1].id, "Backend Engineer (Go)", "Indeed", "https://indeed.com/jobs/2001", "Remote-first, Go + K8s", "active", 80},
		{newID(), companies[1].id, "Senior Backend Engineer", "Referral", "", "Referred by Sarah Chen", "active", 45},
		{newID(), companies[2].id, "Full-Stack Developer", "LinkedIn", "https://linkedin.com/jobs/3001", "React + Node stack", "active", 75},
		{newID(), companies[2].id, "Frontend Engineer", "AngelList", "https://angel.co/datapulse/frontend", "Design-focused role", "archived", 90},
		{newID(), companies[3].id, "Software Engineer II", "Company Website", "https://greenbyte.dev/careers", "Green tech mission", "active", 70},
		{newID(), companies[3].id, "DevOps Engineer", "LinkedIn", "https://linkedin.com/jobs/4002", "Terraform + AWS focus", "archived", 88},
		{newID(), companies[4].id, "ML Engineer", "Hacker News", "https://quantumlabs.ai/jobs/ml", "PyTorch, transformers research", "active", 65},
		{newID(), companies[4].id, "Senior Software Engineer - AI", "Company Website", "https://quantumlabs.ai/jobs/swe-ai", "LLM infra work", "active", 50},
		{newID(), companies[5].id, "Backend Engineer - Payments", "LinkedIn", "https://linkedin.com/jobs/6001", "Payments domain, Go + gRPC", "active", 55},
		{newID(), companies[5].id, "Senior Full-Stack Engineer", "Indeed", "https://indeed.com/jobs/6002", "React + Go, equity package", "active", 40},
		{newID(), companies[6].id, "Frontend Engineer - React", "AngelList", "https://angel.co/pixelcraft/react", "Creative tooling, WebGL", "active", 72},
		{newID(), companies[6].id, "UI Engineer", "LinkedIn", "https://linkedin.com/jobs/7002", "Design systems team", "archived", 85},
		{newID(), companies[7].id, "Platform Engineer", "Referral", "", "Referred by Mike Torres", "active", 35},
		{newID(), companies[7].id, "SRE", "Indeed", "https://indeed.com/jobs/8002", "On-call rotation, good comp", "active", 68},
		{newID(), companies[0].id, "Engineering Manager", "LinkedIn", "https://linkedin.com/jobs/1003", "People management + IC hybrid", "active", 25},
		{newID(), companies[2].id, "Data Engineer", "Company Website", "https://datapulse.io/careers/data-eng", "Spark, Airflow, dbt", "active", 30},
		{newID(), companies[4].id, "Research Engineer", "Hacker News", "", "Published papers preferred", "archived", 95},
		{newID(), companies[5].id, "VP of Engineering", "Referral", "", "Leadership role, pre-IPO equity", "active", 20},
	}

	for _, j := range jobs {
		_, err = tx.Exec(ctx,
			`INSERT INTO jobs (id, user_id, company_id, title, source, url, notes, status, board_column, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'wishlist', $9, $9)`,
			j.id, userID, j.companyID, j.title, j.source, j.url, j.notes, j.status, daysAgo(j.daysAgo),
		)
		must(err, "create job "+j.title)
	}
	fmt.Printf("created %d jobs\n", len(jobs))

	// ── tag some companies and jobs ──────────────────────────────────────
	tagRelations := []struct{ tagID, entityType, entityID string }{
		{tags[0].id, "company", companies[1].id}, // CloudScale = remote
		{tags[1].id, "company", companies[0].id}, // TechNova = onsite
		{tags[2].id, "company", companies[3].id}, // GreenByte = hybrid
		{tags[4].id, "company", companies[4].id}, // Quantum Labs = FAANG-level
		{tags[5].id, "company", companies[6].id}, // PixelCraft = startup
		{tags[5].id, "company", companies[5].id}, // FinEdge = startup
		{tags[8].id, "company", companies[4].id}, // Quantum Labs = good-comp
		{tags[6].id, "job", jobs[3].id},           // Referral job
		{tags[6].id, "job", jobs[14].id},          // Referral job
		{tags[3].id, "job", jobs[0].id},           // high-priority
		{tags[3].id, "job", jobs[9].id},           // high-priority
		{tags[7].id, "job", jobs[8].id},           // interesting-tech ML
		{tags[7].id, "job", jobs[9].id},           // interesting-tech AI
		{tags[0].id, "job", jobs[2].id},           // remote job
	}
	for _, tr := range tagRelations {
		_, err = tx.Exec(ctx,
			`INSERT INTO tag_relations (id, tag_id, entity_type, entity_id, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			newID(), tr.tagID, tr.entityType, tr.entityID, daysAgo(90),
		)
		must(err, "create tag relation")
	}
	fmt.Printf("created %d tag relations\n", len(tagRelations))

	// ── 7. applications ──────────────────────────────────────────────────
	// Each application: job index, resume index, name, status, applied_days_ago, stages to create
	type appDef struct {
		jobIdx    int
		resumeIdx int
		name      string
		status    string
		appliedDA int      // days ago
		stages    []int    // indices into stages slice (how far they progressed)
		stageEnd  []string // final status for each stage: completed, active, pending, skipped, cancelled
	}

	appDefs := []appDef{
		// ── ACTIVE apps (still in pipeline) ──
		{0, 0, "TechNova - Senior SWE", "active", 82, []int{0, 1, 2}, []string{"completed", "completed", "active"}},
		{2, 0, "CloudScale - Backend Go", "active", 78, []int{0, 1}, []string{"completed", "active"}},
		{4, 2, "DataPulse - Full-Stack", "active", 72, []int{0, 1, 2, 3}, []string{"completed", "completed", "completed", "active"}},
		{6, 0, "GreenByte - SWE II", "active", 68, []int{0}, []string{"active"}},
		{8, 0, "Quantum Labs - ML Eng", "active", 62, []int{0, 1, 2}, []string{"completed", "completed", "active"}},
		{10, 0, "FinEdge - Backend Payments", "active", 52, []int{0, 1}, []string{"completed", "active"}},
		{14, 2, "InfraCore - Platform Eng", "active", 32, []int{0, 1, 2}, []string{"completed", "completed", "active"}},
		{17, 2, "DataPulse - Data Eng", "active", 28, []int{0}, []string{"active"}},

		// ── ON HOLD ──
		{1, 0, "TechNova - Staff Platform", "on_hold", 58, []int{0, 1, 2}, []string{"completed", "completed", "completed"}},
		{9, 0, "Quantum Labs - SWE AI", "on_hold", 48, []int{0, 1}, []string{"completed", "completed"}},
		{16, 0, "TechNova - Eng Manager", "on_hold", 23, []int{0}, []string{"completed"}},

		// ── REJECTED ──
		{5, 1, "DataPulse - Frontend", "rejected", 88, []int{0, 1}, []string{"completed", "cancelled"}},
		{7, 0, "GreenByte - DevOps", "rejected", 85, []int{0, 1, 2}, []string{"completed", "completed", "cancelled"}},
		{13, 1, "PixelCraft - UI Eng", "rejected", 83, []int{0}, []string{"cancelled"}},
		{18, 0, "Quantum Labs - Research Eng", "rejected", 92, []int{0, 1, 2}, []string{"completed", "completed", "cancelled"}},

		// ── OFFER ──
		{3, 0, "CloudScale - Senior Backend", "offer", 43, []int{0, 1, 2, 3, 4, 5}, []string{"completed", "completed", "completed", "completed", "completed", "completed"}},
		{11, 2, "FinEdge - Senior FS", "offer", 38, []int{0, 1, 2, 4, 5}, []string{"completed", "completed", "completed", "completed", "completed"}},

		// ── ARCHIVED ──
		{12, 1, "PixelCraft - Frontend React", "archived", 70, []int{0, 1, 2}, []string{"completed", "completed", "skipped"}},
		{15, 0, "InfraCore - SRE", "archived", 66, []int{0, 1}, []string{"completed", "skipped"}},
		{19, 0, "FinEdge - VP Eng", "archived", 18, []int{0}, []string{"skipped"}},
	}

	type appRecord struct{ id, name, status string; jobIdx int }
	var appRecords []appRecord
	type stageRecord struct{ id, appID, stageTemplID, status string; order int }
	var stageRecords []stageRecord

	for _, ad := range appDefs {
		appID := newID()
		appliedAt := daysAgo(ad.appliedDA)
		appRecords = append(appRecords, appRecord{appID, ad.name, ad.status, ad.jobIdx})

		// insert application first (without current_stage_id) so stages can reference it
		_, err = tx.Exec(ctx,
			`INSERT INTO applications (id, user_id, job_id, resume_id, name, current_stage_id, status, applied_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, NULL, $6, $7, $8, $8)`,
			appID, userID, jobs[ad.jobIdx].id, resumes[ad.resumeIdx].id, ad.name, ad.status, appliedAt, appliedAt,
		)
		must(err, "create application "+ad.name)

		// create application stages
		var currentStageID *string
		for i, stageIdx := range ad.stages {
			stageID := newID()
			stStatus := ad.stageEnd[i]
			order := i + 1

			startedAt := appliedAt.Add(time.Duration(i*3+randBetween(0, 5)) * 24 * time.Hour)
			var completedAt *time.Time
			if stStatus == "completed" || stStatus == "skipped" || stStatus == "cancelled" {
				t := startedAt.Add(time.Duration(randBetween(1, 7)) * 24 * time.Hour)
				completedAt = &t
			}

			_, err = tx.Exec(ctx,
				`INSERT INTO application_stages (id, application_id, stage_template_id, status, "order", started_at, completed_at, created_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				stageID, appID, stages[stageIdx].id, stStatus, order, startedAt, completedAt, startedAt,
			)
			must(err, fmt.Sprintf("create stage %s for app %s", stages[stageIdx].name, ad.name))

			stageRecords = append(stageRecords, stageRecord{stageID, appID, stages[stageIdx].id, stStatus, order})

			if stStatus == "active" || stStatus == "pending" || i == len(ad.stages)-1 {
				currentStageID = &stageID
			}
		}

		// update application with current_stage_id
		if currentStageID != nil {
			_, err = tx.Exec(ctx,
				`UPDATE applications SET current_stage_id = $1 WHERE id = $2`,
				*currentStageID, appID,
			)
			must(err, "update current_stage_id for "+ad.name)
		}
	}
	fmt.Printf("created %d applications with stages\n", len(appDefs))

	// ── tag some applications ────────────────────────────────────────────
	appTagRelations := []struct{ tagIdx, appIdx int }{
		{3, 0},  // high-priority -> TechNova SWE
		{3, 4},  // high-priority -> Quantum ML
		{0, 1},  // remote -> CloudScale Backend
		{6, 15}, // referral -> CloudScale Senior (offer)
		{6, 6},  // referral -> InfraCore Platform
		{8, 15}, // good-comp -> CloudScale offer
		{8, 16}, // good-comp -> FinEdge offer
		{5, 2},  // startup -> DataPulse FS
		{9, 19}, // backup -> FinEdge VP archived
		{7, 4},  // interesting-tech -> Quantum ML
	}
	for _, atr := range appTagRelations {
		_, err = tx.Exec(ctx,
			`INSERT INTO tag_relations (id, tag_id, entity_type, entity_id, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			newID(), tags[atr.tagIdx].id, "application", appRecords[atr.appIdx].id, daysAgo(80),
		)
		must(err, "create app tag relation")
	}
	fmt.Printf("created %d application tag relations\n", len(appTagRelations))

	// ── 8. comments ──────────────────────────────────────────────────────
	type commentDef struct {
		appIdx  int
		stageID *string // nil = application-level comment
		content string
		daysAgo int
	}

	var commentDefs []commentDef

	// Application-level comments
	commentDefs = append(commentDefs,
		commentDef{0, nil, "Really excited about this role. The team is working on cutting-edge ML infrastructure.", 80},
		commentDef{0, nil, "Heard back from recruiter, scheduling screening call.", 75},
		commentDef{1, nil, "Remote Go position, exactly what I'm looking for. Applied through Indeed.", 76},
		commentDef{2, nil, "Full-stack role with modern tech stack. Company growing fast.", 70},
		commentDef{4, nil, "Dream role - ML engineering with transformers. Need to brush up on PyTorch.", 60},
		commentDef{4, nil, "Completed the coding challenge, felt pretty good about it.", 50},
		commentDef{6, nil, "Referred by Mike, should have an edge here.", 30},
		commentDef{8, nil, "Staff role might be a stretch but worth trying. Good learning opportunity.", 56},
		commentDef{9, nil, "LLM infrastructure work is exactly my interest area.", 46},
		commentDef{11, nil, "Got the automated rejection email. No feedback provided.", 82},
		commentDef{12, nil, "Rejection after technical round. Feedback: need more system design experience.", 75},
		commentDef{13, nil, "Quick rejection, probably didn't match their requirements.", 81},
		commentDef{15, nil, "Offer received! $185k base + equity. Need to negotiate.", 15},
		commentDef{15, nil, "Counter-offered $200k, waiting to hear back.", 12},
		commentDef{16, nil, "Offer from FinEdge too! $175k + significant pre-IPO equity.", 10},
		commentDef{17, nil, "Decided not to pursue further - not the right fit culturally.", 62},
		commentDef{18, nil, "Withdrew application, focusing on active opportunities.", 60},
		commentDef{19, nil, "VP role is too senior for where I am right now. Archived.", 16},
		commentDef{10, nil, "Put on hold while I evaluate the two offers.", 20},
	)

	// Stage-level comments (find relevant stages)
	for _, sr := range stageRecords {
		for _, ar := range appRecords {
			if sr.appID != ar.id {
				continue
			}

			// Add stage comments for certain combos
			switch {
			case ar.name == "TechNova - Senior SWE" && sr.order == 2:
				commentDefs = append(commentDefs, commentDef{0, &sr.id, "Screening call went well. Recruiter was friendly, discussed comp range $170-200k.", 72})
			case ar.name == "TechNova - Senior SWE" && sr.order == 3:
				commentDefs = append(commentDefs, commentDef{0, &sr.id, "Technical interview scheduled for next Tuesday. Need to review system design patterns.", 65})
			case ar.name == "CloudScale - Backend Go" && sr.order == 2:
				commentDefs = append(commentDefs, commentDef{1, &sr.id, "Phone screen with hiring manager. Team uses Go, gRPC, K8s. Very aligned with my skills.", 70})
			case ar.name == "DataPulse - Full-Stack" && sr.order == 3:
				commentDefs = append(commentDefs, commentDef{2, &sr.id, "3-hour technical interview. Covered React, Node, and SQL. Whiteboard coding went smoothly.", 58})
			case ar.name == "DataPulse - Full-Stack" && sr.order == 4:
				commentDefs = append(commentDefs, commentDef{2, &sr.id, "Take-home: build a small dashboard app. Given 5 days to complete.", 52})
			case ar.name == "Quantum Labs - ML Eng" && sr.order == 3:
				commentDefs = append(commentDefs, commentDef{4, &sr.id, "ML-focused interview. Questions about attention mechanisms and model optimization. Tough but fair.", 48})
			case ar.name == "CloudScale - Senior Backend" && sr.order == 5:
				commentDefs = append(commentDefs, commentDef{15, &sr.id, "Final round with CTO. Great conversation about distributed systems architecture.", 22})
			case ar.name == "CloudScale - Senior Backend" && sr.order == 6:
				commentDefs = append(commentDefs, commentDef{15, &sr.id, "Verbal offer! Will get the written one by EOW.", 18})
			case ar.name == "FinEdge - Senior FS" && sr.order == 5:
				commentDefs = append(commentDefs, commentDef{16, &sr.id, "Offer discussion. Equity details look promising given the IPO timeline.", 14})
			case ar.name == "GreenByte - DevOps" && sr.order == 3:
				commentDefs = append(commentDefs, commentDef{12, &sr.id, "Failed the infrastructure coding exercise. Need to practice more Terraform.", 76})
			case ar.name == "InfraCore - Platform Eng" && sr.order == 3:
				commentDefs = append(commentDefs, commentDef{6, &sr.id, "Technical deep dive on Kubernetes operators. Felt confident about my answers.", 22})
			}
			break
		}
	}

	for _, cd := range commentDefs {
		_, err = tx.Exec(ctx,
			`INSERT INTO comments (id, user_id, application_id, stage_id, content, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
			newID(), userID, appRecords[cd.appIdx].id, cd.stageID, cd.content, daysAgo(cd.daysAgo),
		)
		must(err, "create comment")
	}
	fmt.Printf("created %d comments\n", len(commentDefs))

	// ── 9. reminders ─────────────────────────────────────────────────────
	type reminderDef struct {
		appIdx    int
		message   string
		remindAt  time.Time
		isDone    bool
		createdDA int
	}

	reminderDefs := []reminderDef{
		{0, "Follow up with TechNova recruiter about technical round results", daysAgo(-2), false, 5},
		{2, "Submit DataPulse take-home assignment", daysAgo(48), true, 52},
		{4, "Prepare for Quantum Labs ML interview - review transformer architecture", daysAgo(46), true, 50},
		{6, "Send thank-you email to InfraCore interviewer", daysAgo(-1), false, 3},
		{15, "Respond to CloudScale offer by Friday", daysAgo(-3), false, 8},
		{16, "Compare FinEdge vs CloudScale offers - make decision", daysAgo(-5), false, 6},
		{1, "Check CloudScale backend opening status", daysAgo(60), true, 70},
		{5, "Follow up on FinEdge payment team screening", now.Add(48 * time.Hour), false, 2},
	}

	for _, rd := range reminderDefs {
		_, err = tx.Exec(ctx,
			`INSERT INTO reminders (id, user_id, application_id, stage_id, remind_at, message, is_done, created_at, updated_at)
			 VALUES ($1, $2, $3, NULL, $4, $5, $6, $7, $7)`,
			newID(), userID, appRecords[rd.appIdx].id, rd.remindAt, rd.message, rd.isDone, daysAgo(rd.createdDA),
		)
		must(err, "create reminder")
	}
	fmt.Printf("created %d reminders\n", len(reminderDefs))

	// ── 10. reviewer account (for Chrome Web Store review) ──────────────
	const reviewerEmail = "reviewer@jobber.dev"
	const reviewerPassword = "ReviewerPass2026"
	_, _ = tx.Exec(ctx, `DELETE FROM users WHERE email = $1`, reviewerEmail)

	reviewerID := newID()
	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, email, name, password_hash, locale, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		reviewerID, reviewerEmail, "Chrome Reviewer", hashPassword(reviewerPassword), "en", now,
	)
	must(err, "create reviewer user")

	// Add minimal data so the reviewer can test the full flow
	reviewerStages := []struct{ name string; order int }{
		{"Applied", 1}, {"Interview", 2}, {"Offer", 3},
	}
	for _, s := range reviewerStages {
		_, err = tx.Exec(ctx,
			`INSERT INTO stage_templates (id, user_id, name, "order", created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $5)`,
			newID(), reviewerID, s.name, s.order, now,
		)
		must(err, "create reviewer stage "+s.name)
	}
	fmt.Printf("created reviewer: %s / %s\n", reviewerEmail, reviewerPassword)

	// ── commit ───────────────────────────────────────────────────────────
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}

	fmt.Println("\n✓ seed completed successfully!")
	fmt.Printf("  login: %s / password123\n", seedEmail)
	fmt.Printf("  reviewer: %s / %s\n", reviewerEmail, reviewerPassword)
}

func must(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
