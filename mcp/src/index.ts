import { McpServer } from "@modelcontextprotocol/server";
import { StdioServerTransport } from "@modelcontextprotocol/server/stdio";
import { spawn } from "child_process";
import { join } from "path";
import z from "zod";

const NewMigrationOutput = z.object({
    up: z.string(),
    down: z.string()
});
type NewMigrationOutput = z.infer<typeof NewMigrationOutput>;

async function createMigration(name: string, dir: string, signal: AbortSignal) {
    return new Promise<NewMigrationOutput>((resolve, reject) => {
        const proc = spawn('migrate', ['create', '-ext', 'sql', '-dir', dir, '-seq', name], { signal, cwd: process.cwd() })
        proc.on('exit', (code, signal) => {
            if (code !== 0) {
                reject(new Error('Process ended with exit code ' + code + ', ' + signal));
                return;
            }
            resolve({
                down: join(dir, `${name}.down.sql`),
                up: join(dir, `${name}.up.sql`)
            });
        })
    });
}

const GetMigrationVersionOutput = z.object({
    version: z.coerce.number().describe('The current migration version applied')
});
type GetMigrationVersionOutput = z.infer<typeof GetMigrationVersionOutput>;

async function getCurrentMigration(dir: string, connString: string, signal: AbortSignal) {
    return new Promise<GetMigrationVersionOutput>((resolve, reject) => {
        const proc = spawn('migrate', ['-path', dir, '-database', connString, 'version'], { signal });
        const errors = Array<string>();
        const texts = Array<string>();
        proc.stdout.on('data', chunk => {
            texts.push(String(chunk))
        })
        proc.stderr.on('data', chunk => {
            errors.push(String(chunk));
        })
        proc.on('exit', (code, signal) => {
            if (code !== 0) {
                reject(new Error('Process ended with exit code: ' + code + ', ' + signal));
                return
            }
            try {
                const [version] = texts;
                return GetMigrationVersionOutput.parse({ version });
            } catch (e) {
                if (e instanceof RangeError) {
                    reject(new Error('No database version could be found'))
                    return;
                }
                reject(new Error('An error occurred: ' + e))
            }
        });
    });
}

async function applyMigrations(connString: string, dir: string, signal: AbortSignal, direction: 'up' | 'down', count?: number) {
    const args = [
        '-database',
        connString,
        '-path',
        dir,
        direction
    ];
    if (count !== undefined) {
        args.push(String(count));
    }
    return new Promise<void>((resolve, reject) => {
        const proc = spawn('migrate', args, { signal });
        proc.on('error', reject);
        proc.on('exit', (code, signal) => {
            if (code !== 0) {
                reject(new Error('Process ended with exit code: ' + code + ', ' + signal));
                return
            }
            resolve();
        });
    })
}

async function generateQueries(signal: AbortSignal) {
    return new Promise<void>((resolve, reject) => {
        const proc = spawn('sqlc', ['generate'], { signal });
        proc.on('error', reject);
        proc.on('exit', (code, signal) => {
            if (code !== 0) {
                reject(new Error('Process ended with exit code: ' + code + ', ' + signal));
                return
            }
            resolve();
        });
    })
}

const server = new McpServer({
    name: 'mogtade',
    version: '1.0.0',
});
const migrationsDir = join(process.cwd(), 'infra', 'scripts', 'migrations');

server.registerTool('apply_down_migrations', {
    title: 'Apply down Migrations',
    description: 'Apply backward migrations to the database',
    inputSchema: z.object({
        count: z.number().describe('The number of migrations to apply. This field is required')
    })
}, async ({ count }, { mcpReq }) => {
    try {
        const dbUrl = process.env.DB_URL;
        if (!dbUrl) {
            return { content: [{ type: 'text', text: 'Internal error. Database not found' }], isError: true };
        }
        await applyMigrations(dbUrl, migrationsDir, mcpReq.signal, 'down', count);
        return { content: [{ type: 'text', text: 'Migrations applied successfully' }], };
    } catch (e) {
        return { content: [{ type: 'text', text: (e as Error).message }], isError: true };
    }
})

server.registerTool('apply_forward_migration', {
    title: 'Apply up Migrations',
    description: 'Apply forward migrations to the database',
    inputSchema: z.object({
        count: z.number().optional().describe('The number of migrations to apply. Leave empty to apply all remaining migrations')
    }),
}, async ({ count }, { mcpReq }) => {
    try {
        const dbUrl = process.env.DB_URL;
        if (!dbUrl) {
            return { content: [{ type: 'text', text: 'Internal error. Database not found' }], isError: true };
        }
        await applyMigrations(dbUrl, migrationsDir, mcpReq.signal, 'up', count);
        return { content: [{ type: 'text', text: 'Migrations applied successfully' }], };
    } catch (e) {
        return { content: [{ type: 'text', text: (e as Error).message }], isError: true };
    }
})

server.registerTool('get_migration_version', {
    title: 'Get Version',
    description: 'Gets the current migration version from the database',
    outputSchema: GetMigrationVersionOutput
}, async ({ mcpReq }) => {
    try {
        const dbUrl = process.env.DB_URL;
        if (!dbUrl) {
            return { content: [{ type: 'text', text: 'Internal error. Database not found' }], isError: true };
        }
        const result = await getCurrentMigration(migrationsDir, dbUrl, mcpReq.signal);
        return { content: [{ type: 'text', text: JSON.stringify(result) }], structuredContent: result };
    } catch (e) {
        return { content: [{ type: 'text', text: (e as Error).message }], isError: true };
    }
})

server.registerTool('create_migration', {
    title: 'Create Migration',
    description: 'Creates a pair of migration PostgreSQL up/down scripts',
    inputSchema: z.object({
        name: z.string().regex(/^[a-zA-Z0-9_-]+$/).describe('The name of the migration')
    }) as any,
    outputSchema: NewMigrationOutput
}, async ({ name }, { mcpReq }) => {
    try {
        const result = await createMigration(name, migrationsDir, mcpReq.signal);
        return { content: [{ type: 'text', text: JSON.stringify(result) }], structuredContent: result };
    } catch (e) {
        return { content: [{ type: 'text', text: (e as Error).message }], isError: true };
    }
})

async function main() {
    const transport = new StdioServerTransport();
    await server.connect(transport);
    console.error('Mogtrade MCP server is running on stdio');
}

try {
    await main();
} catch (e) {
    console.error(`Fatal error in main(): ${e}`);
    process.exit(1);
}