CREATE TABLE workflow_entity (
    id character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    active boolean NOT NULL,
    "isArchived" boolean DEFAULT false NOT NULL,
    nodes json NOT NULL,
    connections json NOT NULL,
    settings json,
    "staticData" json,
    "pinData" json,
    meta json,
    "versionId" character varying(255) DEFAULT '0'::character varying NOT NULL,
    "activeVersionId" character varying(255),
    "createdAt" timestamp without time zone NOT NULL,
    "updatedAt" timestamp without time zone NOT NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX "IDX_8d5c4af59af94a50d2e8ba6c63" ON workflow_entity (name);

CREATE TABLE credentials_entity (
    id character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(255) NOT NULL,
    data text NOT NULL,
    "nodesAccess" json NOT NULL,
    "createdAt" timestamp without time zone NOT NULL,
    "updatedAt" timestamp without time zone NOT NULL,
    PRIMARY KEY (id)
);

CREATE UNIQUE INDEX "IDX_53198de30c6a5d44439c0e5a95" ON credentials_entity (name);
