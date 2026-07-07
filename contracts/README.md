# Event Contract Registry

This directory contains the authoritative JSON Schemas for all domain events in the Endurance ecosystem.

## Standard

All events must follow the topic structure:
`domain.entity.action.vVersion`

Example: `checkout.order.created.v1`

## Registry

| Topic | Schema File | Owner |
|-------|-------------|-------|
| `reverse.return.created.v1` | `logistics/reverse.return.created.v1.json` | Logistics (Reverse Hub) |
| `reverse.return.authorized.v1` | `logistics/reverse.return.authorized.v1.json` | Logistics (Reverse Hub) |
| `checkout.order.created.v1` | `checkout/checkout.order.created.v1.json` | Sales (Unified Commerce) |
