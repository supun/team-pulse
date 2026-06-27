import React, { useEffect, useState } from "https://esm.sh/react@18";
import { createRoot } from "https://esm.sh/react-dom@18/client";
import htm from "https://esm.sh/htm@3.1.1";

const html = htm.bind(React.createElement);
const apiBaseUrl = window.__APP_CONFIG__?.API_BASE_URL || "http://localhost:8080";

const architectureCards = [
  {
    title: "What Team Pulse Does",
    body:
      "Team and club management, event scheduling, attendance, messaging, payments, volunteer coordination, and a mobile-first flow where quick RSVP and reminders matter more than deep desktop workflows.",
  },
  {
    title: "Hardest Feature",
    body:
      "Notifications are the hardest because they combine fan-out, delivery guarantees, user preferences, quiet hours, retries, deduplication, and multiple providers. Payments and recurring events are tricky, but notification correctness breaks trust fastest.",
  },
  {
    title: "Event Scheduling Service",
    body:
      "REST endpoints validate time zones, expand recurring events into series instances, return paginated activity feeds, and use ETags plus short-lived caching to reduce mobile payload churn.",
  },
  {
    title: "Notification Platform",
    body:
      "Push, email, and SMS should enter a queue, be processed by background workers, retried with backoff, and written with idempotency keys so duplicate jobs do not trigger duplicate sends.",
  },
  {
    title: "Scaling Pattern",
    body:
      "For millions of users, separate synchronous API latency from asynchronous delivery. Horizontally scale stateless API nodes, queue fan-out work, shard workers by channel or tenant, and cache hot reads close to users.",
  },
  {
    title: "Backend API Focus",
    body:
      "Invitation APIs need versioning and backward compatibility. Mobile clients benefit from pagination, combined list-plus-dashboard responses, caching headers, and fewer round trips than separate endpoints per widget.",
  },
];

const emptyForm = {
  title: "",
  kind: "training",
  startsAt: "",
  timeZone: "Europe/Oslo",
  location: "",
  maxParticipants: 16,
  notes: "",
  recurrenceFrequency: "weekly",
  recurrenceInterval: 1,
  recurrenceCount: 1,
};

const emptyChargeForm = {
  teamId: "team-pulse",
  teamName: "Pulse United",
  plan: "club",
};

const emptyInvitationForm = {
  channel: "push",
  recipients: "mia@example.com, alex@example.com",
  message: "New event available. Please respond in the app.",
};

function App() {
  const [activities, setActivities] = useState([]);
  const [dashboard, setDashboard] = useState(null);
  const [subscriptions, setSubscriptions] = useState([]);
  const [invitations, setInvitations] = useState([]);
  const [selectedActivityId, setSelectedActivityId] = useState("");
  const [memberName, setMemberName] = useState("Mia");
  const [form, setForm] = useState({
    ...emptyForm,
    startsAt: new Date(Date.now() + 86400000).toISOString().slice(0, 16),
  });
  const [chargeForm, setChargeForm] = useState(emptyChargeForm);
  const [invitationForm, setInvitationForm] = useState(emptyInvitationForm);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [checkoutUrl, setCheckoutUrl] = useState("");

  async function loadData(nextSelectedActivityId) {
    const [activitiesRes, subscriptionsRes] = await Promise.all([
      fetch(
        `${apiBaseUrl}/api/v1/events?teamId=team-pulse&limit=20&include=dashboard`
      ),
      fetch(`${apiBaseUrl}/api/v1/subscriptions`),
    ]);

    const activitiesData = await activitiesRes.json();
    const subscriptionsData = await subscriptionsRes.json();

    setActivities(activitiesData.items || []);
    setDashboard(activitiesData.dashboard || null);
    setSubscriptions(subscriptionsData.items || []);

    const resolvedActivityId =
      nextSelectedActivityId || selectedActivityId || activitiesData.items?.[0]?.id || "";
    setSelectedActivityId(resolvedActivityId);

    if (resolvedActivityId) {
      const invitationRes = await fetch(
        `${apiBaseUrl}/api/v1/activities/${resolvedActivityId}/invitations?limit=10`
      );
      const invitationData = await invitationRes.json();
      setInvitations(invitationData.items || []);
    } else {
      setInvitations([]);
    }
  }

  useEffect(() => {
    loadData().catch(() => setError("Unable to load project data."));
  }, []);

  useEffect(() => {
    if (!selectedActivityId) {
      return;
    }
    fetch(`${apiBaseUrl}/api/v1/activities/${selectedActivityId}/invitations?limit=10`)
      .then((response) => response.json())
      .then((payload) => setInvitations(payload.items || []))
      .catch(() => setError("Unable to load invitations."));
  }, [selectedActivityId]);

  async function createActivity(event) {
    event.preventDefault();
    setError("");
    setMessage("");
    setCheckoutUrl("");

    const recurrenceCount = Number(form.recurrenceCount);
    const endpoint = recurrenceCount > 1 ? "/api/v1/event-series" : "/api/v1/events";
    const response = await fetch(`${apiBaseUrl}${endpoint}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        teamId: "team-pulse",
        title: form.title,
        kind: form.kind,
        startsAt: new Date(form.startsAt).toISOString(),
        timeZone: form.timeZone,
        location: form.location,
        maxParticipants: Number(form.maxParticipants),
        notes: form.notes,
        recurrence:
          recurrenceCount > 1
            ? {
                frequency: form.recurrenceFrequency,
                interval: Number(form.recurrenceInterval),
                count: recurrenceCount,
              }
            : null,
      }),
    });

    const payload = await response.json();
    if (!response.ok) {
      setError(payload.error || "Failed to create activity.");
      return;
    }

    setMessage(
      recurrenceCount > 1
        ? `Recurring series created from ${payload.title}.`
        : `Activity created: ${payload.title}.`
    );
    setForm({
      ...emptyForm,
      startsAt: new Date(Date.now() + 86400000).toISOString().slice(0, 16),
    });
    await loadData(payload.firstOccurrenceId || payload.id);
  }

  async function submitRSVP(activityId, status) {
    setError("");
    setMessage("");
    const response = await fetch(`${apiBaseUrl}/api/v1/activities/${activityId}/rsvps`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ memberName, status }),
    });

    const payload = await response.json();
    if (!response.ok) {
      setError(payload.error || "Failed to save RSVP.");
      return;
    }

    await loadData(activityId);
  }

  async function createInvitation(event) {
    event.preventDefault();
    setError("");
    setMessage("");
    if (!selectedActivityId) {
      setError("Select an activity before sending invitations.");
      return;
    }

    const response = await fetch(
      `${apiBaseUrl}/api/v1/activities/${selectedActivityId}/invitations`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": self.crypto?.randomUUID?.() || `${Date.now()}`,
        },
        body: JSON.stringify({
          channel: invitationForm.channel,
          recipients: invitationForm.recipients
            .split(",")
            .map((value) => value.trim())
            .filter(Boolean),
          message: invitationForm.message,
        }),
      }
    );

    const payload = await response.json();
    if (!response.ok) {
      setError(payload.error || "Failed to queue invitations.");
      return;
    }

    setMessage(`Invitation queued via ${payload.channel} for ${payload.recipients.length} recipient(s).`);
    await loadData(selectedActivityId);
  }

  async function startStripeCheckout(event) {
    event.preventDefault();
    setError("");
    setMessage("");

    const response = await fetch(`${apiBaseUrl}/api/v1/checkout-sessions`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ...chargeForm,
        successUrl: `${window.location.origin}/billing/success`,
        cancelUrl: `${window.location.origin}/billing/cancel`,
      }),
    });

    const payload = await response.json();
    if (!response.ok) {
      setError(payload.error || "Failed to create Stripe checkout session.");
      return;
    }

    setMessage(`Stripe Checkout session created for ${payload.subscription.teamName}.`);
    setCheckoutUrl(payload.checkoutUrl || "");
    await loadData(selectedActivityId);
  }

  return html`
    <main className="shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Full-stack sports coordination demo</p>
          <h1>TeamPulse</h1>
          <p className="lede">
            A small sports coordination project for planning matches, training,
            volunteer shifts and social events with a Go API and a mobile-first React UI.
          </p>
        </div>
        <div className="hero-card">
          <label>RSVP as</label>
          <input
            value=${memberName}
            onChange=${(event) => setMemberName(event.target.value)}
            placeholder="Member name"
          />
          <p>
            The feed now uses the versioned mobile API, supports recurring events,
            invitation delivery, pagination, and short-lived caching.
          </p>
        </div>
      </section>

      ${dashboard &&
      html`
        <section className="stats">
          <article>
            <span>Upcoming</span>
            <strong>${dashboard.upcomingActivities}</strong>
          </article>
          <article>
            <span>Total going</span>
            <strong>${dashboard.totalGoing}</strong>
          </article>
          <article>
            <span>Next event</span>
            <strong>${dashboard.nextActivity ? dashboard.nextActivity.title : "None"}</strong>
          </article>
          <article>
            <span>Subscriptions</span>
            <strong>${subscriptions.length}</strong>
          </article>
        </section>
      `}

      <section className="architecture-grid">
        ${architectureCards.map(
          (card) => html`
            <article className="insight-card" key=${card.title}>
              <h3>${card.title}</h3>
              <p>${card.body}</p>
            </article>
          `
        )}
      </section>

      <section className="grid">
        <div className="stack">
          <form className="panel" onSubmit=${createActivity}>
            <div className="panel-heading">
              <h2>Create activity</h2>
              <p>Recurring series, time-zone validation, and mobile-friendly payloads.</p>
            </div>

            <label>
              Title
              <input
                value=${form.title}
                onChange=${(event) => setForm({ ...form, title: event.target.value })}
                required
              />
            </label>

            <label>
              Type
              <select
                value=${form.kind}
                onChange=${(event) => setForm({ ...form, kind: event.target.value })}
              >
                <option value="match">Match</option>
                <option value="training">Training</option>
                <option value="volunteer">Volunteer</option>
                <option value="social">Social</option>
              </select>
            </label>

            <label>
              Starts at
              <input
                type="datetime-local"
                value=${form.startsAt}
                onChange=${(event) => setForm({ ...form, startsAt: event.target.value })}
                required
              />
            </label>

            <label>
              Time zone
              <select
                value=${form.timeZone}
                onChange=${(event) => setForm({ ...form, timeZone: event.target.value })}
              >
                <option value="Europe/Oslo">Europe/Oslo</option>
                <option value="Europe/London">Europe/London</option>
                <option value="UTC">UTC</option>
                <option value="America/New_York">America/New_York</option>
              </select>
            </label>

            <label>
              Location
              <input
                value=${form.location}
                onChange=${(event) => setForm({ ...form, location: event.target.value })}
                required
              />
            </label>

            <label>
              Max participants
              <input
                type="number"
                min="1"
                value=${form.maxParticipants}
                onChange=${(event) => setForm({ ...form, maxParticipants: event.target.value })}
                required
              />
            </label>

            <label>
              Recurrence count
              <input
                type="number"
                min="1"
                max="24"
                value=${form.recurrenceCount}
                onChange=${(event) => setForm({ ...form, recurrenceCount: event.target.value })}
              />
            </label>

            <label>
              Recurrence frequency
              <select
                value=${form.recurrenceFrequency}
                onChange=${(event) =>
                  setForm({ ...form, recurrenceFrequency: event.target.value })
                }
              >
                <option value="weekly">Weekly</option>
                <option value="daily">Daily</option>
              </select>
            </label>

            <label>
              Recurrence interval
              <input
                type="number"
                min="1"
                value=${form.recurrenceInterval}
                onChange=${(event) =>
                  setForm({ ...form, recurrenceInterval: event.target.value })
                }
              />
            </label>

            <label>
              Notes
              <textarea
                value=${form.notes}
                onChange=${(event) => setForm({ ...form, notes: event.target.value })}
              ></textarea>
            </label>

            <button type="submit">Create activity</button>
          </form>

          <form className="panel" onSubmit=${createInvitation}>
            <div className="panel-heading">
              <h2>Queue invitations</h2>
              <p>Invitation API with idempotency protection and channel selection.</p>
            </div>

            <label>
              Activity
              <select
                value=${selectedActivityId}
                onChange=${(event) => setSelectedActivityId(event.target.value)}
              >
                ${activities.map(
                  (activity) => html`
                    <option value=${activity.id} key=${activity.id}>
                      ${activity.title}
                    </option>
                  `
                )}
              </select>
            </label>

            <label>
              Channel
              <select
                value=${invitationForm.channel}
                onChange=${(event) =>
                  setInvitationForm({ ...invitationForm, channel: event.target.value })
                }
              >
                <option value="push">Push</option>
                <option value="email">Email</option>
                <option value="sms">SMS</option>
              </select>
            </label>

            <label>
              Recipients
              <textarea
                value=${invitationForm.recipients}
                onChange=${(event) =>
                  setInvitationForm({ ...invitationForm, recipients: event.target.value })
                }
              ></textarea>
            </label>

            <label>
              Message
              <textarea
                value=${invitationForm.message}
                onChange=${(event) =>
                  setInvitationForm({ ...invitationForm, message: event.target.value })
                }
              ></textarea>
            </label>

            <button type="submit">Send invitations</button>
          </form>

          <form className="panel" onSubmit=${startStripeCheckout}>
            <div className="panel-heading">
              <h2>Start Stripe checkout</h2>
              <p>Creates a Stripe subscription checkout session in the payment service.</p>
            </div>

            <label>
              Team ID
              <input
                value=${chargeForm.teamId}
                onChange=${(event) =>
                  setChargeForm({ ...chargeForm, teamId: event.target.value })
                }
                required
              />
            </label>

            <label>
              Team name
              <input
                value=${chargeForm.teamName}
                onChange=${(event) =>
                  setChargeForm({ ...chargeForm, teamName: event.target.value })
                }
                required
              />
            </label>

            <label>
              Plan
              <select
                value=${chargeForm.plan}
                onChange=${(event) =>
                  setChargeForm({ ...chargeForm, plan: event.target.value })
                }
              >
                <option value="starter">Starter</option>
                <option value="club">Club</option>
                <option value="pro">Pro</option>
              </select>
            </label>

            <button type="submit">Create checkout session</button>
          </form>

          ${(error || message) &&
          html`
            <section className="panel feedback">
              ${error && html`<p className="error">${error}</p>`}
              ${message && html`<p className="success">${message}</p>`}
              ${checkoutUrl &&
              html`<p className="success">Checkout URL: <a href=${checkoutUrl}>${checkoutUrl}</a></p>`}
            </section>
          `}
        </div>

        <section className="panel">
          <div className="panel-heading">
            <h2>Upcoming activities</h2>
            <p>Each card supports quick RSVP interactions and shows recurrence metadata.</p>
          </div>

          <div className="activity-list">
            ${activities.map(
              (activity) => html`
                <article className="activity-card" key=${activity.id}>
                  <div className="activity-meta">
                    <span className="kind">${activity.kind}</span>
                    <span>
                      ${new Date(activity.startsAt).toLocaleString([], {
                        dateStyle: "medium",
                        timeStyle: "short",
                      })}
                    </span>
                    <span>${activity.timeZone}</span>
                  </div>
                  <h3>${activity.title}</h3>
                  <p>${activity.location}</p>
                  <p className="notes">${activity.notes}</p>
                  ${activity.recurrence &&
                  html`
                    <p className="notes">
                      ${activity.recurrence.frequency} series, occurrence
                      ${activity.occurrenceIndex}${activity.seriesId ? ` in ${activity.seriesId}` : ""}
                    </p>
                  `}
                  <div className="attendance">
                    <span>Going ${activity.goingCount}</span>
                    <span>Maybe ${activity.maybeCount}</span>
                    <span>Declined ${activity.declinedCount}</span>
                  </div>
                  <div className="actions">
                    <button onClick=${() => submitRSVP(activity.id, "going")}>Going</button>
                    <button onClick=${() => submitRSVP(activity.id, "maybe")}>Maybe</button>
                    <button onClick=${() => submitRSVP(activity.id, "declined")}>
                      Decline
                    </button>
                  </div>
                </article>
              `
            )}
          </div>

          <div className="panel-heading subscriptions-heading">
            <h2>Queued invitations</h2>
            <p>Current activity: ${selectedActivityId || "None selected"}.</p>
          </div>
          <div className="activity-list">
            ${invitations.map(
              (invitation) => html`
                <article className="activity-card" key=${invitation.id}>
                  <div className="activity-meta">
                    <span className="kind">${invitation.channel}</span>
                    <span>${invitation.status}</span>
                  </div>
                  <h3>${invitation.id}</h3>
                  <p>${invitation.recipients.join(", ")}</p>
                  <p className="notes">${invitation.message}</p>
                </article>
              `
            )}
            ${!invitations.length && html`<p className="notes">No queued invitations for this activity yet.</p>`}
          </div>

          <div className="panel-heading subscriptions-heading">
            <h2>Subscriptions</h2>
            <p>Billing state is isolated in the payment microservice.</p>
          </div>
          <div className="activity-list">
            ${subscriptions.map(
              (subscription) => html`
                <article className="activity-card" key=${subscription.teamId}>
                  <div className="activity-meta">
                    <span className="kind">${subscription.plan}</span>
                    <span>${subscription.status}</span>
                  </div>
                  <h3>${subscription.teamName}</h3>
                  <p>${subscription.teamId}</p>
                  <p className="notes">
                    NOK ${subscription.priceMonthlyNOK}/month, renews
                    ${new Date(subscription.renewalDate).toLocaleDateString()}.
                  </p>
                  <div className="attendance">
                    <span>Last payment ${subscription.lastPaymentStatus}</span>
                    ${subscription.stripeSessionId &&
                    html`<span>Stripe session ${subscription.stripeSessionId}</span>`}
                  </div>
                </article>
              `
            )}
          </div>
        </section>
      </section>
    </main>
  `;
}

createRoot(document.getElementById("root")).render(html`<${App} />`);
