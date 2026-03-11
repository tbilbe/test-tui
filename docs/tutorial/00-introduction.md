# Building Terminal UI Applications in Go

A comprehensive tutorial for building production-ready TUI applications, using the Seven Test TUI as our reference implementation.

## Who This Is For

This tutorial assumes you:
- Have basic Go syntax knowledge (variables, functions, if/else)
- Understand what a terminal is

By the end, you'll understand:
- How to structure a Go application professionally
- The Elm Architecture pattern used by modern TUI frameworks
- AWS SDK integration patterns
- State management in event-driven applications

## The Application We're Building

The Seven Test TUI is a tool for QA testers to manipulate football fixture data without needing to understand DynamoDB or backend APIs. It features:

- Cognito authentication
- Multiple screens (login → prefix selection → gameweeks → fixtures)
- Real-time data editing
- Direct database writes

## Chapter Overview

| Chapter | Topic | Level |
|---------|-------|-------|
| 01 | Project Setup & Go Modules | Junior |
| 02 | Understanding Structs | Junior |
| 03 | The Entry Point Pattern | Junior |
| 04 | Configuration Management | Junior |
| 05 | Introduction to Bubbletea | Junior → Mid |
| 06 | The Elm Architecture | Mid |
| 07 | Building Your First Screen | Mid |
| 08 | Commands & Async Operations | Mid |
| 09 | Multi-Screen Navigation | Mid |
| 10 | AWS SDK Integration | Mid → Senior |
| 11 | API Client Patterns | Mid → Senior |
| 12 | State Management | Senior |
| 13 | Testing TUI Applications | Senior |
| 14 | **Bonus**: Extending the Application | All Levels |

## How to Use This Tutorial

Each chapter builds on the previous. Code examples show the evolution from nothing to the final application.

When you see code blocks, they're meant to be typed out, not copied. Muscle memory matters.

Let's begin.

---

[Next: Chapter 1 - Project Setup →](./01-project-setup.md)
