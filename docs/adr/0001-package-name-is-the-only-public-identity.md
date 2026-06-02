# Package Name Is The Only Public Identity

Agent Skill Registry exposes the npm-compatible **Package Name** as the only public identity for an Agent Skill in V1. We rejected separate public skill IDs and `skills.sh`-style `{source}/{skill}` aliases because they would introduce competing identities, complicate installation, import, governance, and search semantics, and undermine the single-skill package model.
