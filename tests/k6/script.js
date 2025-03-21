// tests/k6/script.js
import http from 'k6/http';
import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { randomItem } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// ------------------------------------------------------------------------
// 1) MEETS WITH ADMIN CREDENTIALS
//    Each meet has its own admin account (director).
//    Fill in real credentials for each meet accordingly.
// ------------------------------------------------------------------------
const MEETS_WITH_ADMINS = [
    { meetName: 'APL-State-Meet',    adminUser: 'adminApl',   adminPass: 'passApl' },
    { meetName: 'BigBench2025',      adminUser: 'adminBb',    adminPass: 'passBb' },
    { meetName: 'Winter-Classic',    adminUser: 'adminWinter',adminPass: 'passWinter' },
    { meetName: 'Regionals',         adminUser: 'adminReg',   adminPass: 'passReg' },
    { meetName: 'Saturday-Open',     adminUser: 'adminSat',   adminPass: 'passSat' },
    { meetName: 'Junior-Champs',     adminUser: 'adminJr',    adminPass: 'passJr' },
    { meetName: 'Deadlift-Only',     adminUser: 'adminDlo',   adminPass: 'passDlo' },
    { meetName: 'All-Women-Classic', adminUser: 'adminAwc',   adminPass: 'passAwc' },
    { meetName: 'Masters-Invitational', adminUser: 'adminMas',pass: 'passMas' },
    { meetName: 'Open-Class2025',    adminUser: 'adminOpen',  adminPass: 'passOpen' },
];

// 2) Normal user credentials (for protected endpoints).
//    Replace or expand as you see fit.
const NORMAL_USERS = [
    { username: 'alice', password: 'alicePass' },
    { username: 'bob',   password: 'bobPass' },
];

// 3) Referee positions
const REF_POSITIONS = ['left', 'center', 'right'];

// 4) Basic config
const BASE_URL = 'https://referee-lights.michaelkingston.com.au';

// ------------------------------------------------------------------------
// 5) K6 OPTIONS
// ------------------------------------------------------------------------
export let options = {
    thresholds: {
        http_req_failed: ['rate<0.01'],     // <1% errors
        http_req_duration: ['p(95)<2000'], // 95% requests < 2s
    },
    scenarios: {
        // Public scenario: hits public endpoints (no auth required).
        publicScenario: {
            executor: 'constant-arrival-rate',
            rate: 5,              // 5 iterations per minute
            timeUnit: '1m',
            duration: '5m',
            preAllocatedVUs: 3,
            maxVUs: 5,
            exec: 'publicFlow',
        },
        // Protected scenario: logs in as a normal user, does a few steps.
        protectedScenario: {
            executor: 'constant-arrival-rate',
            rate: 5,
            timeUnit: '1m',
            duration: '5m',
            preAllocatedVUs: 3,
            maxVUs: 5,
            exec: 'protectedFlow',
        },
        // Director/admin scenario: logs in as admin for a random meet,
        // performs forceVacate or resetInstance, logs out.
        directorScenario: {
            executor: 'constant-arrival-rate',
            rate: 2,
            timeUnit: '1m',
            duration: '5m',
            preAllocatedVUs: 3,
            maxVUs: 5,
            exec: 'directorFlow',
        },
        // Referee scenario: scans the QR code path, then opens WebSocket to
        // submit a decision or startTimer.
        refereeScenario: {
            executor: 'constant-arrival-rate',
            rate: 10,
            timeUnit: '1m',
            duration: '5m',
            preAllocatedVUs: 5,
            maxVUs: 10,
            exec: 'refereeFlow',
        },
    },
};

// ------------------------------------------------------------------------
// SCENARIO A: PUBLIC FLOW (NO AUTH)
//   - For example, we fetch / (ShowMeets) or /health, /index, etc.
// ------------------------------------------------------------------------
export function publicFlow() {
    // 1) GET /
    let res1 = http.get(`${BASE_URL}/`);
    check(res1, { 'GET / success': (r) => r.status === 200 });

    // 2) GET /health
    let res2 = http.get(`${BASE_URL}/health`);
    check(res2, { 'health check': (r) => r.status === 200 });

    // 3) GET /index (though typically you might need a meet set. But let's see.)
    let res3 = http.get(`${BASE_URL}/index`);
    check(res3, { 'GET /index': (r) => [200, 302, 404].includes(r.status) });

    sleep(1);
}

// ------------------------------------------------------------------------
// SCENARIO B: PROTECTED FLOW (NORMAL USER AUTH)
//   - logs in, claims a position, maybe logs out
// ------------------------------------------------------------------------
export function protectedFlow() {
    // 1) Pick a meet from MEETS_WITH_ADMINS (just for variety)
    let meetPick = randomItem(MEETS_WITH_ADMINS);
    let meetName = meetPick.meetName;
    // 2) pick a normal user from our array
    let user = randomItem(NORMAL_USERS);

    // set-meet
    let setMeetRes = http.post(`${BASE_URL}/set-meet`, { meetName });
    check(setMeetRes, { 'set-meet status': (r) => [200, 302].includes(r.status) });

    // login as normal user
    let loginRes = http.post(`${BASE_URL}/login?meetName=${meetName}`, {
        username: user.username,
        password: user.password,
    });
    check(loginRes, { 'protected login': (r) => [200, 302].includes(r.status) });

    // claim a position
    let pos = randomItem(REF_POSITIONS);
    let claimRes = http.post(`${BASE_URL}/position/claim`, { position: pos });
    check(claimRes, { 'claimed position': (r) => [200, 302].includes(r.status) });

    // vacate
    let vacateRes = http.post(`${BASE_URL}/position/vacate`, {});
    check(vacateRes, { 'vacate position': (r) => [200, 302].includes(r.status) });

    // logout
    let logoutRes = http.post(`${BASE_URL}/logout`, {});
    check(logoutRes, { 'logout': (r) => [200, 302].includes(r.status) });

    sleep(1);
}

// ------------------------------------------------------------------------
// SCENARIO C: DIRECTOR FLOW (ADMIN ROUTES)
//   - logs in as admin for a specific meet, does forceVacate or resetInstance
// ------------------------------------------------------------------------
export function directorFlow() {
    // pick a random meet + admin creds
    let pick = randomItem(MEETS_WITH_ADMINS);
    let meetName = pick.meetName;

    // 1) set-meet
    let setMeetRes = http.post(`${BASE_URL}/set-meet`, { meetName });
    check(setMeetRes, { 'set-meet success': (r) => [200, 302].includes(r.status) });

    // 2) login as admin
    let loginRes = http.post(`${BASE_URL}/login?meetName=${meetName}`, {
        username: pick.adminUser,
        password: pick.adminPass,
    });
    check(loginRes, { 'admin login': (r) => [200, 302].includes(r.status) });

    // 3) random admin action
    let adminActions = [
        {
            name: 'forceVacateLeft',
            path: '/admin/force-vacate',
            body: { meetName, position: 'left' },
        },
        {
            name: 'resetInstance',
            path: '/admin/reset-instance',
            body: { meetName },
        },
    ];
    let chosen = randomItem(adminActions);
    let adminRes = http.post(`${BASE_URL}${chosen.path}`, chosen.body);
    check(adminRes, { 'admin action': (r) => [200, 302].includes(r.status) });

    // 4) logout
    let logoutRes = http.post(`${BASE_URL}/logout`, {});
    check(logoutRes, { 'admin logout': (r) => [200, 302].includes(r.status) });

    sleep(1);
}

// ------------------------------------------------------------------------
// SCENARIO D: REFEREE FLOW (WEBSOCKET)
//   - /referee/:meetName/:position, then open wss://... for startTimer or submitDecision
// ------------------------------------------------------------------------
export function refereeFlow() {
    // pick a random meet from MEETS_WITH_ADMINS
    let pick = randomItem(MEETS_WITH_ADMINS);
    let meetName = pick.meetName;

    // pick a random position
    let position = randomItem(REF_POSITIONS);

    // first do GET /referee/:meetName/:position
    let refereeUrl = `${BASE_URL}/referee/${meetName}/${position}`;
    let refRes = http.get(refereeUrl);
    check(refRes, { 'referee seat claim': (r) => [200, 302].includes(r.status) });

    // now open the WebSocket
    let wsUrl = `wss://referee-lights.michaelkingston.com.au/referee-updates?meetName=${meetName}`;
    let doStartTimer = Math.random() < 0.5;
    let sentCount = 0;
    let receivedCount = 0;

    let conn = ws.connect(wsUrl, {}, function (socket) {
        socket.on('open', () => {
            if (doStartTimer) {
                console.log(`[WS] startTimer => meet=${meetName}, pos=${position}`);
                socket.send(JSON.stringify({ action: 'startTimer', meetName }));
                sentCount++;
            } else {
                let decision = Math.random() < 0.5 ? 'good' : 'bad';
                console.log(`[WS] submitDecision => meet=${meetName}, pos=${position}`);
                socket.send(JSON.stringify({
                    action: 'submitDecision',
                    meetName,
                    judgeId: position,
                    decision,
                }));
                sentCount++;
            }
        });

        socket.on('message', (msg) => {
            receivedCount++;
            console.log(`[WS] message: ${msg}`);
        });

        socket.on('error', (err) => {
            console.error(`[WS] error => ${err.error()}`);
        });

        // close after 2 seconds
        setTimeout(() => {
            console.log(`[WS] closing => meet=${meetName}, sent=${sentCount}, received=${receivedCount}`);
            socket.close();
        }, 2000);
    });

    check(conn, { 'WS connected (101)': (r) => r && r.status === 101 });
    sleep(1);
}
