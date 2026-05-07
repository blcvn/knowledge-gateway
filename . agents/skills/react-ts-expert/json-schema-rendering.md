---
skill_id: SKILL-007
version: 1.0.0
status: active
priority: existing
group: Frontend Development
created_at: 2026-04-24
---

# SKILL-007 · Dynamic UI Schema Rendering (JSON Schema → React)

## Mô tả

Kỹ năng chuyên biệt render React components động từ JSON Schema — Component Factory pattern, dynamic form generation, wizard/stepper rendering.

## Agents sử dụng

- `ui-renderer-agent`

---

## Năng lực cốt lõi

### 1. Component Factory Pattern

```tsx
// JSON Schema → React Component mapping
interface UISchema {
  type: 'form' | 'table' | 'dashboard' | 'detail' | 'wizard'
  title: string
  fields?: FieldSchema[]
  columns?: ColumnSchema[]
  steps?: StepSchema[]
  actions?: ActionSchema[]
}

interface FieldSchema {
  id: string
  type: 'text' | 'textarea' | 'select' | 'checkbox' | 'date' | 'file' | 'rich-text'
  label: string
  required?: boolean
  validation?: ValidationRule[]
  options?: SelectOption[]  // for 'select' type
  dependsOn?: DependencyRule  // conditional visibility
}

// Component Factory
function renderComponent(schema: UISchema): React.ReactElement {
  switch (schema.type) {
    case 'form':      return <DynamicForm schema={schema} />
    case 'table':     return <DynamicTable schema={schema} />
    case 'dashboard': return <DynamicDashboard schema={schema} />
    case 'detail':    return <DynamicDetail schema={schema} />
    case 'wizard':    return <DynamicWizard schema={schema} />
    default:
      throw new Error(`Unknown schema type: ${schema.type}`)
  }
}

// Field renderer
function renderField(field: FieldSchema, control: Control): React.ReactElement {
  const fieldMap = {
    'text':      DynamicTextField,
    'textarea':  DynamicTextarea,
    'select':    DynamicSelect,
    'checkbox':  DynamicCheckbox,
    'date':      DynamicDatePicker,
    'file':      DynamicFileUpload,
    'rich-text': DynamicRichText,
  } as const

  const Component = fieldMap[field.type]
  if (!Component) throw new Error(`Unknown field type: ${field.type}`)

  return <Component key={field.id} field={field} control={control} />
}
```

### 2. JSON Schema Form (react-hook-form + Zod)

```tsx
// Dynamic form with validation from schema
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'

function buildZodSchema(fields: FieldSchema[]): z.ZodObject<any> {
  const shape: Record<string, z.ZodTypeAny> = {}
  
  fields.forEach(field => {
    let validator: z.ZodTypeAny = z.string()
    
    if (field.type === 'text' || field.type === 'textarea') {
      validator = z.string()
      field.validation?.forEach(rule => {
        if (rule.type === 'minLength') validator = (validator as z.ZodString).min(rule.value)
        if (rule.type === 'maxLength') validator = (validator as z.ZodString).max(rule.value)
        if (rule.type === 'pattern')   validator = (validator as z.ZodString).regex(new RegExp(rule.value))
      })
    }
    
    if (!field.required) validator = validator.optional()
    shape[field.id] = validator
  })
  
  return z.object(shape)
}

function DynamicForm({ schema }: { schema: UISchema }) {
  const zodSchema = buildZodSchema(schema.fields ?? [])
  
  const { control, handleSubmit, formState: { errors } } = useForm({
    resolver: zodResolver(zodSchema),
  })
  
  return (
    <form onSubmit={handleSubmit(data => console.log(data))}>
      <h1>{schema.title}</h1>
      {schema.fields?.map(field => renderField(field, control))}
      {schema.actions?.map(action => (
        <Button key={action.id} type={action.type} variant={action.variant}>
          {action.label}
        </Button>
      ))}
    </form>
  )
}
```

### 3. Dynamic Navigation (KG → Navigation)

```tsx
// Knowledge Graph workflow transitions → React navigation
interface NavigationSchema {
  screens: ScreenNode[]
  transitions: Transition[]
  currentScreen: string
}

interface Transition {
  from: string
  to: string
  trigger: 'action' | 'auto' | 'conditional'
  condition?: string  // e.g., "form.isValid && user.isAdmin"
  label?: string
}

function DynamicNavigation({ nav }: { nav: NavigationSchema }) {
  const [current, setCurrent] = useState(nav.currentScreen)
  
  const availableTransitions = nav.transitions
    .filter(t => t.from === current && evaluateCondition(t.condition))
  
  return (
    <nav>
      {availableTransitions.map(t => (
        <button key={t.to} onClick={() => setCurrent(t.to)}>
          {t.label ?? `Go to ${t.to}`}
        </button>
      ))}
    </nav>
  )
}
```

### 4. Wizard / Stepper Rendering

```tsx
interface StepSchema {
  id: string
  title: string
  fields: FieldSchema[]
  validation?: 'immediate' | 'on-next'
}

function DynamicWizard({ schema }: { schema: UISchema & { steps: StepSchema[] } }) {
  const [currentStep, setCurrentStep] = useState(0)
  const [completedData, setCompletedData] = useState<Record<string, any>>({})
  const totalSteps = schema.steps.length
  
  const step = schema.steps[currentStep]
  
  const handleNext = (stepData: Record<string, any>) => {
    setCompletedData(prev => ({ ...prev, [step.id]: stepData }))
    if (currentStep < totalSteps - 1) {
      setCurrentStep(prev => prev + 1)
    }
  }
  
  return (
    <div>
      {/* Progress indicator */}
      <StepIndicator steps={schema.steps} current={currentStep} />
      
      {/* Current step form */}
      <DynamicForm
        schema={{ ...schema, fields: step.fields, title: step.title }}
        onSubmit={handleNext}
      />
      
      {/* Navigation */}
      <div>
        {currentStep > 0 && (
          <Button onClick={() => setCurrentStep(p => p - 1)}>Back</Button>
        )}
        <Button type="submit">
          {currentStep === totalSteps - 1 ? 'Submit' : 'Next'}
        </Button>
      </div>
    </div>
  )
}
```

### 5. Code Export

```tsx
// Generate clean React code from schema
function generateReactCode(schema: UISchema): string {
  const componentName = toPascalCase(schema.title)
  
  const imports = [
    "import React from 'react'",
    "import { useForm } from 'react-hook-form'",
    schema.fields?.some(f => f.required) ? "import { z } from 'zod'" : null,
  ].filter(Boolean).join('\n')
  
  const zodSchemaCode = generateZodSchema(schema.fields ?? [])
  const formCode = generateFormJSX(schema)
  
  return `${imports}

${zodSchemaCode}

export function ${componentName}() {
  const { register, handleSubmit } = useForm()
  
  return (
    ${formCode}
  )
}
`
}
```

---

## JSON Schema Examples

```json
// Sample UI Schema for Document Upload form
{
  "type": "form",
  "title": "Upload Requirement Document",
  "fields": [
    {
      "id": "title",
      "type": "text",
      "label": "Document Title",
      "required": true,
      "validation": [
        { "type": "minLength", "value": 3 },
        { "type": "maxLength", "value": 200 }
      ]
    },
    {
      "id": "project_id",
      "type": "select",
      "label": "Project",
      "required": true,
      "options": [
        { "value": "proj-001", "label": "EKMS Demo" }
      ]
    },
    {
      "id": "content",
      "type": "file",
      "label": "PRD File (Markdown)",
      "required": true
    }
  ],
  "actions": [
    { "id": "submit", "type": "submit", "label": "Upload & Process", "variant": "primary" },
    { "id": "cancel", "type": "button", "label": "Cancel", "variant": "secondary" }
  ]
}
```

## Checklist

- [ ] Component Factory handles all field types defined in schema
- [ ] Zod schema generated dynamically from field validation rules
- [ ] Conditional field visibility (dependsOn) implemented
- [ ] Wizard steps validate before proceeding to next step
- [ ] Empty states handled for all component types
- [ ] Accessibility: form labels linked to inputs, ARIA attributes
- [ ] Generated code is clean and production-ready (no debug code)
