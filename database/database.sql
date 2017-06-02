CREATE TABLE "projects" (
	"id" integer NOT NULL,
	"last_scanned_id" VARCHAR(255),
	CONSTRAINT projects_pk PRIMARY KEY ("id")
);

CREATE TABLE "results" (
	"id" serial NOT NULL,
	"project" integer NOT NULL,
	"commit" VARCHAR(255) NOT NULL,
	"path" VARCHAR(255) NOT NULL,
	"caption" TEXT NOT NULL,
	"description" TEXT NOT NULL,
	CONSTRAINT results_pk PRIMARY KEY ("id")
);

ALTER TABLE "results" ADD CONSTRAINT "results_fk0" FOREIGN KEY ("project") REFERENCES "projects"("id");
