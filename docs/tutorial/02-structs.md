# Chapter 2: Understanding Structs

## What You'll Learn

- What structs are and why Go uses them
- How to define and use structs
- Struct tags and their purpose
- Pointer vs value receivers
- The difference between `*int` and `int`

---

## What is a Struct?

A struct is a collection of fields. It's how Go groups related data together.

```go
type Person struct {
    Name string
    Age  int
}
```

If you're coming from other languages:
- **JavaScript**: Like an object with a fixed shape
- **Python**: Like a class with only data attributes
- **Java/C#**: Like a class, but no inheritance

---

## Our First Domain Model

Let's build the `GameWeek` struct from our application. Create `internal/models/models.go`:

```go
package models

type GameWeek struct {
    GameWeekID        string
    Label             string
    FixturesStartDate string
    FixturesEndDate   string
}
```

### Using the Struct

```go
// Create a GameWeek
gw := GameWeek{
    GameWeekID: "1",
    Label:      "Week 1",
}

// Access fields
fmt.Println(gw.Label)  // "Week 1"

// Modify fields
gw.Label = "Week 1 - Updated"
```

### Zero Values

In Go, uninitialized fields get "zero values":

```go
gw := GameWeek{}
fmt.Println(gw.GameWeekID)  // "" (empty string)
fmt.Println(gw.Label)       // "" (empty string)
```

| Type | Zero Value |
|------|------------|
| string | "" |
| int | 0 |
| bool | false |
| pointer | nil |
| slice | nil |

---

## Struct Tags

Look at the actual `GameWeek` in our codebase:

```go
type GameWeek struct {
    GameWeekID        string `json:"gameWeekId" dynamodbav:"gameWeekId"`
    Label             string `json:"label" dynamodbav:"label"`
    FixturesStartDate string `json:"fixturesStartDate" dynamodbav:"fixturesStartDate"`
}
```

Those backtick strings are **struct tags**. They're metadata that other packages can read.

### What Tags Do

**`json:"gameWeekId"`** - When encoding/decoding JSON, use "gameWeekId" as the field name:

```go
// Without tag: {"GameWeekID": "1"}
// With tag:    {"gameWeekId": "1"}
```

**`dynamodbav:"gameWeekId"`** - Same idea, but for DynamoDB's attribute value marshalling.

### Why This Matters

APIs typically use camelCase (`gameWeekId`), but Go convention is PascalCase (`GameWeekID`). Tags bridge this gap.

One struct, multiple serialization formats:

```go
gw := GameWeek{GameWeekID: "1", Label: "Week 1"}

// To JSON (for API)
jsonBytes, _ := json.Marshal(gw)
// {"gameWeekId":"1","label":"Week 1"}

// To DynamoDB (for database)
av, _ := attributevalue.MarshalMap(gw)
// map[gameWeekId:{S: "1"} label:{S: "Week 1"}]
```

---

## Optional Fields with Pointers

Look at this field:

```go
type GameWeek struct {
    // ...
    Locked *bool `json:"locked,omitempty"`
}
```

### Why `*bool` instead of `bool`?

A `bool` can only be `true` or `false`. But what if the field might not exist at all?

```go
// With bool - can't distinguish "false" from "not set"
type GameWeek struct {
    Locked bool
}
gw := GameWeek{}
fmt.Println(gw.Locked)  // false - but is it "false" or "not set"?

// With *bool - nil means "not set"
type GameWeek struct {
    Locked *bool
}
gw := GameWeek{}
fmt.Println(gw.Locked)  // nil - clearly "not set"
```

### The `omitempty` Tag

```go
Locked *bool `json:"locked,omitempty"`
```

`omitempty` means: "If this field is nil/zero, don't include it in the JSON output."

```go
gw := GameWeek{GameWeekID: "1"}
json.Marshal(gw)
// {"gameWeekId":"1"}  -- no "locked" field

locked := true
gw.Locked = &locked
json.Marshal(gw)
// {"gameWeekId":"1","locked":true}
```

---

## A More Complex Example: Fixture

```go
type Fixture struct {
    FixtureID    string        `json:"fixtureId" dynamodbav:"fixtureId"`
    GameWeekID   string        `json:"gameWeekId" dynamodbav:"gameWeekId"`
    Period       FixturePeriod `json:"period" dynamodbav:"period"`
    ClockTimeMin int           `json:"clockTimeMin" dynamodbav:"clockTimeMin"`
    HomeScore    *int          `json:"homeScore,omitempty" dynamodbav:"homeScore,omitempty"`
    AwayScore    *int          `json:"awayScore,omitempty" dynamodbav:"awayScore,omitempty"`
    Participants Participants  `json:"participants" dynamodbav:"participants"`
    Goals        []Goal        `json:"goals,omitempty" dynamodbav:"goals,omitempty"`
}
```

Notice:
- **`FixturePeriod`** - A custom type (we'll cover this next)
- **`*int` for scores** - Scores might not exist yet (pre-match)
- **`Participants`** - Nested struct
- **`[]Goal`** - Slice of structs

---

## Type Aliases for Enums

Go doesn't have enums like other languages. We use type aliases with constants:

```go
type FixturePeriod string

const (
    PeriodPreMatch   FixturePeriod = "PRE_MATCH"
    PeriodFirstHalf  FixturePeriod = "FIRST_HALF"
    PeriodHalfTime   FixturePeriod = "HALF_TIME"
    PeriodSecondHalf FixturePeriod = "SECOND_HALF"
    PeriodFullTime   FixturePeriod = "FULL_TIME"
)
```

### Why Not Just Use `string`?

Type safety:

```go
// With plain string - compiles but wrong
fixture.Period = "HALFTIME"  // Typo! Should be "HALF_TIME"

// With FixturePeriod - IDE helps, constants are clear
fixture.Period = PeriodHalfTime  // Correct, autocomplete helps
```

You can still assign any string (Go doesn't prevent it), but the constants make correct usage obvious.

---

## Methods on Structs

Structs can have methods:

```go
func (f *Fixture) Validate() error {
    if err := ValidatePeriod(f.Period); err != nil {
        return err
    }
    if err := ValidateScore(f.HomeScore); err != nil {
        return err
    }
    return nil
}
```

### Pointer vs Value Receiver

**`func (f *Fixture)`** - Pointer receiver. The method can modify `f`.

**`func (f Fixture)`** - Value receiver. The method gets a copy, can't modify original.

Rule of thumb:
- Use pointer receiver if the method modifies the struct
- Use pointer receiver for large structs (avoids copying)
- Be consistent - if one method uses pointer, all should

---

## Nested Structs

Structs can contain other structs:

```go
type Participants struct {
    Home Team `json:"home" dynamodbav:"home"`
    Away Team `json:"away" dynamodbav:"away"`
}

type Team struct {
    TeamID   string `json:"teamId" dynamodbav:"teamId"`
    TeamName string `json:"teamName" dynamodbav:"teamName"`
}

type Fixture struct {
    // ...
    Participants Participants `json:"participants" dynamodbav:"participants"`
}
```

Access nested fields:

```go
fixture.Participants.Home.TeamName  // "Arsenal"
```

---

## The Complete Models File

Here's what `internal/models/models.go` looks like:

```go
package models

import (
    "fmt"
    "time"
)

type FixturePeriod string

const (
    PeriodPreMatch   FixturePeriod = "PRE_MATCH"
    PeriodFirstHalf  FixturePeriod = "FIRST_HALF"
    PeriodHalfTime   FixturePeriod = "HALF_TIME"
    PeriodSecondHalf FixturePeriod = "SECOND_HALF"
    PeriodFullTime   FixturePeriod = "FULL_TIME"
)

type GameWeek struct {
    GameWeekID        string `json:"gameWeekId" dynamodbav:"gameWeekId"`
    Label             string `json:"label" dynamodbav:"label"`
    FixturesStartDate string `json:"fixturesStartDate" dynamodbav:"fixturesStartDate"`
    FixturesEndDate   string `json:"fixturesEndDate" dynamodbav:"fixturesEndDate"`
    Locked            *bool  `json:"locked,omitempty" dynamodbav:"locked,omitempty"`
}

type Team struct {
    TeamID   string `json:"teamId" dynamodbav:"teamId"`
    TeamName string `json:"teamName" dynamodbav:"teamName"`
}

type Participants struct {
    Home Team `json:"home" dynamodbav:"home"`
    Away Team `json:"away" dynamodbav:"away"`
}

type Fixture struct {
    FixtureID    string        `json:"fixtureId" dynamodbav:"fixtureId"`
    GameWeekID   string        `json:"gameWeekId" dynamodbav:"gameWeekId"`
    Period       FixturePeriod `json:"period" dynamodbav:"period"`
    ClockTimeMin int           `json:"clockTimeMin" dynamodbav:"clockTimeMin"`
    ClockTimeSec int           `json:"clockTimeSec" dynamodbav:"clockTimeSec"`
    HomeScore    *int          `json:"homeScore,omitempty" dynamodbav:"homeScore,omitempty"`
    AwayScore    *int          `json:"awayScore,omitempty" dynamodbav:"awayScore,omitempty"`
    Participants Participants  `json:"participants" dynamodbav:"participants"`
}

func (f *Fixture) Validate() error {
    if err := ValidatePeriod(f.Period); err != nil {
        return err
    }
    return nil
}

func ValidatePeriod(period FixturePeriod) error {
    switch period {
    case PeriodPreMatch, PeriodFirstHalf, PeriodHalfTime, PeriodSecondHalf, PeriodFullTime:
        return nil
    }
    return fmt.Errorf("invalid period: %s", period)
}
```

---

## Key Takeaways

1. **Structs group related data** - Use them for domain models
2. **Struct tags control serialization** - One struct, multiple formats
3. **Use pointers for optional fields** - `*int` can be nil, `int` cannot
4. **`omitempty` excludes zero values** - Cleaner JSON output
5. **Type aliases create pseudo-enums** - Better than raw strings
6. **Methods belong on structs** - Keep logic with data
7. **Pointer receivers for mutation** - Value receivers for read-only

---

## Exercise

1. Add a `Goal` struct with fields: `GoalID`, `PlayerName`, `TimeMin`, `Period`
2. Add a `Goals []Goal` field to `Fixture`
3. Write a `Validate()` method for `Goal` that checks `TimeMin` is between 0-90
4. What happens if you use `Goals []Goal` vs `Goals []*Goal`?

---

[← Previous: Project Setup](./01-project-setup.md) | [Next: Chapter 3 - The Entry Point Pattern →](./03-entry-point.md)

---

<details>
<summary><strong>Exercise Answers</strong> (click to expand)</summary>

### Exercise 1: Add a Goal struct

```go
type Goal struct {
    GoalID     string        `json:"goalId" dynamodbav:"goalId"`
    PlayerName string        `json:"playerName" dynamodbav:"playerName"`
    TimeMin    int           `json:"timeMin" dynamodbav:"timeMin"`
    Period     FixturePeriod `json:"period" dynamodbav:"period"`
}
```

### Exercise 2: Add Goals field to Fixture

```go
type Fixture struct {
    FixtureID    string        `json:"fixtureId" dynamodbav:"fixtureId"`
    // ... other fields ...
    Goals        []Goal        `json:"goals,omitempty" dynamodbav:"goals,omitempty"`
}
```

### Exercise 3: Validate method for Goal

```go
func (g *Goal) Validate() error {
    if g.TimeMin < 0 || g.TimeMin > 90 {
        return fmt.Errorf("timeMin must be between 0 and 90, got %d", g.TimeMin)
    }
    return nil
}
```

**Important**: A validation method is useless unless you call it. You'd wire it into `Fixture.Validate()`:

```go
func (f *Fixture) Validate() error {
    // ... existing validation ...
    
    // Validate all goals
    for i, goal := range f.Goals {
        if err := goal.Validate(); err != nil {
            return fmt.Errorf("goal %d: %w", i, err)
        }
    }
    return nil
}
```

Then `Fixture.Validate()` gets called in commands before saving to DynamoDB:

```go
func updateFixtureCmd(fixture *models.Fixture, prefix string) tea.Cmd {
    return func() tea.Msg {
        // Validate before saving
        if err := fixture.Validate(); err != nil {
            return fixtureUpdatedMsg{err: fmt.Errorf("validation failed: %w", err)}
        }
        // ... proceed with save
    }
}
```

### Exercise 4: `[]Goal` vs `[]*Goal`

**`Goals []Goal`** (slice of values):
- Each Goal is stored directly in the slice
- Modifying a Goal requires indexing: `fixture.Goals[0].TimeMin = 10`
- Simpler, less indirection
- Better for small structs

**`Goals []*Goal`** (slice of pointers):
- Each element is a pointer to a Goal
- Can have `nil` elements
- Modifications affect the original: `goal := fixture.Goals[0]; goal.TimeMin = 10`
- Better for large structs (avoids copying)
- Required if you need to distinguish "not set" from "zero value"

For this use case, `[]Goal` is simpler and sufficient.

</details>
